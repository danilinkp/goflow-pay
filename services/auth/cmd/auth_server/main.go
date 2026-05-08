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
	"shared/pkg/logger"
	"shared/pkg/logger/sl"
	postgresTrm "shared/pkg/transactor/postgres"
	"syscall"
	"time"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
)

func main() {
	cfg := config.MustLoad()

	start := time.Now()

	log := logger.New(cfg.Env, cfg.Log.Level, cfg.Log.Output, cfg.Log.File)
	log.Info("starting auth service",
		slog.String("env", cfg.Env),
	)

	ctx, stopApp := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stopApp()

	pool, err := postgresPool.NewPool(ctx, cfg.DB.DSN(), cfg.DB.ConnectTimeout, cfg.DB.MaxRetriesTime)
	if err != nil {
		log.Error("failed to connect to db",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)),
		)
		os.Exit(1)
	}
	defer pool.Close()
	err = migrations.RunMigrations(pool)
	if err != nil {
		log.Error("failed to run migrations",
			sl.ErrWithStack(err),
			sl.Duration(time.Since(start)))
		os.Exit(1)
	}
	log.Info("migrations applied")

	redisClient, err := redisdb.NewRedisClient(ctx, cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.ReadTimeout, cfg.Redis.WriteTimeout)
	if err != nil {
		log.Error("failed to connect to redis", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		os.Exit(1)
	}

	jwtManager, err := jwt.NewJWTService(cfg.JWT.PrivateKeyPath, cfg.JWT.TTL)
	if err != nil {
		log.Error("failed to create JWT manager", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
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

	errChan := make(chan error, 1)
	go func() {
		if err = gRPCServer.Run(); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("stopping application...")
	case err = <-errChan:
		log.Error("grpc server failed", "err", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
	}

	gRPCServer.Stop()
	log.Info("gracefully stopped")
}
