package main

import (
	"context"
	appkafka "notifications/internal/app/kafka"
	grpcclient "notifications/internal/clients/grpc"
	"notifications/internal/config"
	"notifications/internal/infrastructure/email"
	"notifications/internal/service"
	mongoRepo "notifications/internal/storage/mongo"
	"notifications/internal/storage/postgres"
	"notifications/migrations"
	"os"
	"os/signal"
	mongoDB "shared/pkg/db/mongo"
	postgresPool "shared/pkg/db/postgres"
	"shared/pkg/logger"
	"shared/pkg/logger/sl"
	"syscall"
	"time"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.MustLoad()

	start := time.Now()
	log := logger.New(cfg.Env, cfg.Log.Level, cfg.Log.Output, cfg.Log.File)

	log.Info("starting notification service", "env", cfg.Env)

	ctx, stopApp := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stopApp()

	var notificationRepo service.NotificationRepository

	switch cfg.Storage.Type {
	case "mongodb":
		mongoClient, err := mongoDB.NewClient(ctx, cfg.Storage.Mongo.URI, cfg.Storage.Mongo.ConnectTimeout)
		if err != nil {
			log.Error("failed to connect to mongodb", sl.Err(err))
			os.Exit(1)
		}
		defer func() { _ = mongoClient.Disconnect(ctx) }()

		db := mongoClient.Database(cfg.Storage.Mongo.Name)

		notificationRepo = mongoRepo.NewNotificationRepo(db)
	default:
		pool, err := postgresPool.NewPool(ctx, cfg.Storage.Postgres.DSN(), cfg.Storage.Postgres.ConnectTimeout, cfg.Storage.Postgres.MaxRetriesTime)
		if err != nil {
			log.Error("failed to connect to db", "err", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
			os.Exit(1)
		}
		defer pool.Close()
		err = migrations.RunMigrations(pool)
		if err != nil {
			log.Error("failed to run migrations", "err", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
			os.Exit(1)
		}
		log.Info("migrations applied")

		getter := trmpgx.DefaultCtxGetter
		notificationRepo = postgres.NewNotificationRepo(pool, getter)
	}

	emailSender := email.NewSmtpSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Password, cfg.SMTP.From)

	conn, err := grpc.NewClient(cfg.AuthGRPC.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("failed to create grpc client", "err", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()
	userClient := grpcclient.NewUserGrpcClient(conn)

	notificationService := service.NewNotificationService(userClient, notificationRepo, emailSender, log)

	kafkaApp := appkafka.NewNotificationApp(log, cfg.Kafka.Brokers(), cfg.Kafka.Topic, notificationService)

	if err = kafkaApp.Run(ctx); err != nil {
		log.Error("app stopped with error", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		os.Exit(1)
	}
	log.Info("notification app stopped")
}
