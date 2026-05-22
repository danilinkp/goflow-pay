package main

import (
	grpcapp "auth/internal/app/grpc"
	"auth/internal/config"
	"auth/internal/infrastructure/jwt"
	"auth/internal/infrastructure/security"
	"auth/internal/services"
	mongoRepo "auth/internal/storage/mongo"
	"auth/internal/storage/postgres"
	"auth/internal/storage/redis"
	"auth/migrations"
	"context"
	"log/slog"
	"os"
	"os/signal"
	mongoDB "shared/pkg/db/mongo"
	postgresPool "shared/pkg/db/postgres"
	redisdb "shared/pkg/db/redis"
	jwtValidator "shared/pkg/jwt"
	"shared/pkg/logger"
	"shared/pkg/logger/sl"
	trm "shared/pkg/transactor"
	trmmongo "shared/pkg/transactor/mongo"
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

	var (
		userRepo    services.UserRepository
		companyRepo services.CompanyRepository
		trmAdapter  trm.Transactioner
	)

	switch cfg.Storage.Type {
	case "mongodb":
		mongoClient, err := mongoDB.NewClient(ctx, cfg.Storage.Mongo.URI, cfg.Storage.Mongo.ConnectTimeout)
		if err != nil {
			log.Error("failed to connect to mongodb", sl.Err(err))
			os.Exit(1)
		}
		defer mongoClient.Disconnect(ctx)

		db := mongoClient.Database(cfg.Storage.Mongo.Name)
		factory, err := mongoRepo.NewRepositoryFactory(ctx, db)
		if err != nil {
			log.Error("failed to init mongodb repositories", sl.Err(err))
			os.Exit(1)
		}
		userRepo = factory.UserRepo()
		companyRepo = factory.CompanyRepo()

		trmAdapter = trmmongo.NewMongoAdapter(mongoClient)

	default:
		pool, err := postgresPool.NewPool(ctx, cfg.Storage.Postgres.DSN(), cfg.Storage.Postgres.ConnectTimeout, cfg.Storage.Postgres.MaxRetriesTime)
		if err != nil {
			log.Error("failed to connect to postgres", sl.Err(err))
			os.Exit(1)
		}
		defer pool.Close()

		if err = migrations.RunMigrations(pool); err != nil {
			log.Error("failed to run migrations", sl.Err(err))
			os.Exit(1)
		}

		getter := trmpgx.DefaultCtxGetter
		userRepo = postgres.NewUserRepo(pool, getter)
		companyRepo = postgres.NewCompanyRepo(pool, getter)

		trManager := manager.Must(trmpgx.NewDefaultFactory(pool))
		trmAdapter = trm.NewAvitoAdapter(trManager)
	}

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
