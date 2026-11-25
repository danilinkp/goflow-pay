package main

import (
	grpcapp "accounts/internal/app/grpc"
	"accounts/internal/config"
	"accounts/internal/infrastructure/bank"
	"accounts/internal/services"
	mongoRepo "accounts/internal/storage/mongo"
	"accounts/internal/storage/postgres"
	"accounts/migrations"
	"context"
	"log/slog"
	"os"
	"os/signal"
	mongoDB "shared/pkg/db/mongo"
	postgresPool "shared/pkg/db/postgres"
	"shared/pkg/logger"
	"shared/pkg/logger/sl"
	"shared/pkg/metrics"
	"shared/pkg/outbox"
	"shared/pkg/outbox/publisher/kafka"
	outboxMongoRepository "shared/pkg/outbox/repository/mongo"
	outboxPgRepository "shared/pkg/outbox/repository/postgres"
	"shared/pkg/tracing"
	trmmongo "shared/pkg/transactor/mongo"
	"sync"
	"syscall"
	"time"

	trm "shared/pkg/transactor"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

func main() {
	cfg := config.MustLoad()
	shutdownTrace, err := tracing.Init(context.Background(), "accounts")
	if err != nil {
		panic(err)
	}
	defer func() { _ = shutdownTrace(context.Background()) }()

	start := time.Now()

	log := logger.New(cfg.Env, cfg.Log.Level, cfg.Log.Output, cfg.Log.File)
	log.Info("starting account service",
		slog.String("env", cfg.Env),
	)

	log.Info("starting account service", "env", cfg.Env, "read_only", cfg.ReadOnly)

	ctx, stopApp := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stopApp()

	var (
		accountRepo          services.AccountRepository
		bankAccountRepo      services.BankAccountRepository
		accountOperationRepo services.AccountOperationRepository
		bankOperationRepo    services.BankOperationRepository
		statementRepo        services.StatementRepository
		outboxRepo           outbox.OutboxRepository
		trmAdapter           trm.Transactioner
	)

	switch cfg.Storage.Type {
	case "mongodb":
		mongoClient, err := mongoDB.NewClient(ctx, cfg.Storage.Mongo.URI, cfg.Storage.Mongo.ConnectTimeout)
		if err != nil {
			log.Error("failed to connect to mongodb", sl.Err(err))
			os.Exit(1)
		}
		defer func() { _ = mongoClient.Disconnect(ctx) }()

		db := mongoClient.Database(cfg.Storage.Mongo.Name)
		repos := mongoRepo.NewRepositories(db)
		err = repos.EnsureIndexes(ctx)
		if err != nil {
			log.Error("failed to init mongodb repositories", sl.Err(err))
			os.Exit(1)
		}
		outboxRepo = outboxMongoRepository.NewOutboxRepo(db)
		if err = outboxRepo.EnsureIndexes(ctx); err != nil {
			log.Error("failed to init mongodb outbox", sl.Err(err))
			os.Exit(1)
		}
		accountRepo = repos.Account
		bankAccountRepo = repos.BankAccount
		accountOperationRepo = repos.AccountOperation
		bankOperationRepo = repos.BankOperation
		statementRepo = repos.Statement

		trmAdapter = trmmongo.NewMongoAdapter(mongoClient)
	default:
		pool, err := postgresPool.NewPool(ctx, cfg.Storage.Postgres.DSN(), cfg.Storage.Postgres.ConnectTimeout, cfg.Storage.Postgres.MaxRetriesTime)
		if err != nil {
			log.Error("failed to connect to db", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
			os.Exit(1)
		}
		defer pool.Close()
		if cfg.ReadOnly {
			log.Info("read-only mode: migrations skipped")
		} else {
			err = migrations.RunMigrations(pool)
			if err != nil {
				log.Error("failed to run migrations", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
				os.Exit(1)
			}
			log.Info("migrations applied")
		}

		getter := trmpgx.DefaultCtxGetter
		accountRepo = postgres.NewAccountRepo(pool, getter)
		bankAccountRepo = postgres.NewBankAccountRepo(pool, getter)
		accountOperationRepo = postgres.NewAccountOperationRepo(pool, getter)
		bankOperationRepo = postgres.NewBankOperationRepo(pool, getter)
		statementRepo = postgres.NewStatementRepo(pool, getter)
		outboxRepo = outboxPgRepository.NewOutboxRepo(pool, getter)

		trManager := manager.Must(trmpgx.NewDefaultFactory(pool))
		trmAdapter = trm.NewAvitoAdapter(trManager)
	}
	bankGateway := bank.NewMockBankGateway(cfg.BankGateway.FailureRate)

	accountsService := services.NewAccountService(accountRepo, accountOperationRepo, bankAccountRepo,
		bankOperationRepo, statementRepo, bankGateway, outboxRepo, trmAdapter, log)

	var wg sync.WaitGroup

	if cfg.ReadOnly {
		log.Info("read-only mode: outbox worker disabled")
	} else {
		pub := kafka.NewPublisher(cfg.Kafka.Brokers())
		defer func() {
			if closeErr := pub.Close(); closeErr != nil {
				log.Error("failed to close kafka publisher", sl.ErrWithStack(closeErr), sl.Duration(time.Since(start)))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			worker := outbox.NewWorker(log, outboxRepo, pub, cfg.Outbox.Interval, cfg.Outbox.BatchSize)
			worker.Run(ctx)
			log.Info("outbox worker stopped")
		}()
	}

	metrics.Serve(log, cfg.MetricsAddr)

	gRPCServer := grpcapp.NewAccountsApp(log, accountsService, cfg.GRPCServer.Port)

	errChan := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := gRPCServer.Run(); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("stopping application...")
	case err := <-errChan:
		log.Error("grpc server failed", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		stopApp()
	}

	gRPCServer.Stop()
	wg.Wait()
	log.Info("gracefully stopped")
}
