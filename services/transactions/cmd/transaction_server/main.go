package main

import (
	"context"
	"os"
	"os/signal"
	postgresPool "shared/pkg/db/postgres"
	"shared/pkg/logger"
	"shared/pkg/logger/sl"
	"shared/pkg/outbox"
	"shared/pkg/outbox/publisher/kafka"
	outboxRepository "shared/pkg/outbox/repository/postgres"
	postgresTrm "shared/pkg/transactor/postgres"
	"sync"
	"syscall"
	"time"
	grpcapp "transactions/internal/app/grpc"
	grpcclient "transactions/internal/clients/grpc"
	"transactions/internal/config"
	"transactions/internal/infrastructure/workers"
	"transactions/internal/services"
	"transactions/internal/storage/postgres"
	"transactions/migrations"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.MustLoad()

	start := time.Now()
	log := logger.New(cfg.Env)

	log.Info("starting transaction service", "env", cfg.Env)

	ctx, stopApp := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stopApp()

	pool, err := postgresPool.NewPool(ctx, cfg.DB.DSN(), cfg.DB.ConnectTimeout, cfg.DB.MaxRetriesTime)
	if err != nil {
		log.Error("failed to connect to db", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		os.Exit(1)
	}
	defer pool.Close()
	err = migrations.RunMigrations(pool)
	if err != nil {
		log.Error("failed to run migrations", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		os.Exit(1)
	}
	log.Info("migrations applied")

	trManager := manager.Must(trmpgx.NewDefaultFactory(pool))
	trmAdapter := postgresTrm.NewTransactionAdapter(trManager)

	getter := trmpgx.DefaultCtxGetter
	transactionRepo := postgres.NewTransactionRepo(pool, getter)
	outboxRepo := outboxRepository.NewOutboxRepo(pool, getter)
	conn, err := grpc.NewClient(cfg.GRPCClient.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("failed to create account grpc client", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		os.Exit(1)
	}
	defer conn.Close()
	accountClient := grpcclient.NewAccountGRPCClient(conn)

	transactionService := services.NewTransactionService(accountClient, transactionRepo, outboxRepo, trmAdapter, log)

	pub := kafka.NewPublisher(cfg.Kafka.Brokers())
	defer func() {
		if closeErr := pub.Close(); closeErr != nil {
			log.Error("failed to close kafka publisher", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker := outbox.NewWorker(log, outboxRepo, pub, cfg.Outbox.Interval, cfg.Outbox.BatchSize)
		worker.Run(ctx)
		log.Info("outbox worker stopped")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		recoverWorker := workers.NewRecoverWorker(transactionRepo, transactionService, cfg.Recover.Interval, cfg.Recover.StaleAfter)
		recoverWorker.Run(ctx)
		log.Info("recover worker stopped")
	}()

	gRPCServer := grpcapp.NewTransactionApp(log, transactionService, cfg.GRPCServer.Port)

	errChan := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err = gRPCServer.Run(); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("stopping application...")
	case err = <-errChan:
		log.Error("grpc server failed", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		stopApp()
	}

	gRPCServer.Stop()
	wg.Wait()
	log.Info("gracefully stopped")
}
