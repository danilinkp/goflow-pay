package main

import (
	grpcapp "accounts/internal/app/grpc"
	"accounts/internal/config"
	"accounts/internal/infrastructure/bank"
	"accounts/internal/services"
	"accounts/internal/storage/postgres"
	"accounts/migrations"
	"context"
	"log/slog"
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

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

func main() {
	cfg := config.MustLoad()

	start := time.Now()

	log := logger.New(cfg.Env)
	log.Info("starting account service",
		slog.String("env", cfg.Env),
	)

	log.Info("starting account service", "env", cfg.Env)

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
	accountRepo := postgres.NewAccountRepo(pool, getter)
	bankAccountRepo := postgres.NewBankAccountRepo(pool, getter)
	accountOperationRepo := postgres.NewAccountOperationRepo(pool, getter)
	bankOperationRepo := postgres.NewBankOperationRepo(pool, getter)
	outboxRepo := outboxRepository.NewOutboxRepo(pool, getter)

	bankGateway := bank.NewMockBankGateway(cfg.BankGateway.FailureRate)

	accountsService := services.NewAccountService(accountRepo, accountOperationRepo, bankAccountRepo, bankOperationRepo, bankGateway, outboxRepo, trmAdapter, log)

	pub := kafka.NewPublisher(cfg.Kafka.Brokers())
	defer func() {
		if closeErr := pub.Close(); closeErr != nil {
			log.Error("failed to close kafka publisher", sl.ErrWithStack(closeErr), sl.Duration(time.Since(start)))
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

	gRPCServer := grpcapp.NewAccountsApp(log, accountsService, cfg.GRPCServer.Port)

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
