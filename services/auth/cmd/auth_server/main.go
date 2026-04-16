package main

import (
	grpcapp "auth/internal/app/grpc"
	"auth/internal/config"
	"auth/internal/infrastructure/jwt"
	"auth/internal/infrastructure/security"
	"auth/internal/services"
	"auth/internal/storage/postgres"
	"auth/internal/storage/redis"
	"auth/migrations"
	"context"
	"log/slog"
	"os"
	"os/signal"
	postgresPool "shared/pkg/db/postgres"
	redisdb "shared/pkg/db/redis"
	jwtValidator "shared/pkg/jwt"
	"shared/pkg/logger/slogpretty"
	postgresTrm "shared/pkg/transactor/postgres"
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
	log.Info("starting auth server", "env", cfg.Env)

	ctx := context.Background()

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

	redisClient, err := redisdb.NewRedisClient(ctx, cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.ReadTimeout, cfg.Redis.WriteTimeout)
	if err != nil {
		log.Error("failed to connect to redis", "err", err)
		os.Exit(1)
	}

	jwtManager, err := jwt.NewJWTService(cfg.JWT.PrivateKeyPath, cfg.JWT.TTL)
	if err != nil {
		log.Error("failed to create JWT manager", "err", err)
		os.Exit(1)
	}
	validator := jwtValidator.NewValidatorFromKey(jwtManager.PublicKey())

	hasher := security.NewBcryptHasher(cfg.HashCost)

	trManager := manager.Must(trmpgx.NewDefaultFactory(pool))
	trmAdapter := postgresTrm.NewTransactionAdapter(trManager)

	getter := trmpgx.DefaultCtxGetter
	userRepo := postgres.NewUserRepo(pool, getter)
	companyRepo := postgres.NewCompanyRepo(pool, getter)
	blackListRepo := redis.NewBlackListRepository(redisClient)

	authService := services.NewAuthService(userRepo, companyRepo, trmAdapter, blackListRepo, jwtManager, validator, hasher, log)

	gRPCServer := grpcapp.NewAuthApp(log, authService, cfg.GRPCServer.Port)

	go func() {
		gRPCServer.MustRun()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	gRPCServer.Stop()
	log.Info("Gracefully stopped")
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
