package main

import (
	"context"
	appkafka "notifications/internal/app/kafka"
	grpcclient "notifications/internal/clients/grpc"
	"notifications/internal/config"
	"notifications/internal/infrastructure/email"
	"notifications/internal/services"
	"notifications/internal/storage/postgres"
	"notifications/migrations"
	"os"
	"os/signal"
	postgresPool "shared/pkg/db/postgres"
	"shared/pkg/logger"
	"shared/pkg/logger/sl"
	"syscall"
	"time"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	start := time.Now()
	log := logger.New(cfg.Env)

	log.Info("starting notification service", "env", cfg.Env)

	ctx, stopApp := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stopApp()

	pool, err := postgresPool.NewPool(ctx, cfg.DB.DSN(), cfg.DB.ConnectTimeout, cfg.DB.MaxRetriesTime)
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
	notificationRepo := postgres.NewNotificationRepo(pool, getter)

	var emailSender services.EmailSender

	if cfg.Env == envLocal {
		emailSender = email.NewMockSender(log)
	} else {
		emailSender = email.NewSmtpSender(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Password, cfg.SMTP.From)
	}

	conn, err := grpc.NewClient(cfg.AuthGRPC.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("failed to create grpc client", "err", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		os.Exit(1)
	}
	defer conn.Close()
	userClient := grpcclient.NewUserGrpcClient(conn)

	notificationService := services.NewNotificationService(userClient, notificationRepo, emailSender, log)

	kafkaApp := appkafka.NewNotificationApp(log, cfg.Kafka.Brokers(), cfg.Kafka.Topic, notificationService)

	if err = kafkaApp.Run(ctx); err != nil {
		log.Error("app stopped with error", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		os.Exit(1)
	}
	log.Info("notification app stopped")
}
