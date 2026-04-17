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
	"shared/pkg/logger/slogpretty"
	"shared/pkg/outbox"
	"shared/pkg/outbox/publisher/kafka"
	outboxRepository "shared/pkg/outbox/repository/postgres"
	postgresTrm "shared/pkg/transactor/postgres"
	"sync"
	"syscall"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)
	log.Info("starting account service", "env", cfg.Env)

	ctx, stopApp := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stopApp()

	pool, err := postgresPool.NewPool(ctx, cfg.DB.DSN(), cfg.DB.ConnectTimeout, cfg.DB.MaxRetriesTime)
	if err != nil {
		log.Error("failed to connect to db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	err = migrations.RunMigrations(pool)
	if err != nil {
		log.Error("failed to run migrations", "err", err)
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
			log.Error("failed to close kafka publisher", "err", closeErr)
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
		log.Error("grpc server failed", "err", err)
		stopApp()
	}

	gRPCServer.Stop()
	wg.Wait()
	log.Info("gracefully stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
