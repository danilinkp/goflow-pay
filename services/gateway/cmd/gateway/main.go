package main

import (
	"context"
	"errors"
	grpcClient "gateway/internal/clients/grpc"
	"gateway/internal/config"
	"gateway/internal/delivery/http/handlers"
	"gateway/internal/delivery/http/router"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	jwtValidator "shared/pkg/jwt"
	"shared/pkg/logger/sl"
	"shared/pkg/logger/slogpretty"
	"syscall"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)
	log.Info("starting gateway service", "env", cfg.Env)

	publicKeyBytes, err := os.ReadFile(cfg.JWTPublicKeyPath)
	if err != nil {
		log.Error("failed to read JWT public key", "err", sl.Err(err))
	}
	validator, err := jwtValidator.NewValidator(publicKeyBytes)
	if err != nil {
		log.Error("failed to create validator", "err", sl.Err(err))
		os.Exit(1)
	}

	clients, err := grpcClient.InitClients(
		cfg.GRPCClients.AuthAddress,
		cfg.GRPCClients.AccountAddress,
		cfg.GRPCClients.TransactionAddress,
	)
	if err != nil {
		log.Error("failed to init grpc clients", "err", sl.Err(err))
		os.Exit(1)
	}
	defer clients.Close()

	authHandler := handlers.NewAuthHandler(clients.Auth)
	systemHandler := handlers.NewSystemHandler(clients.Auth, cfg.BootstrapToken)
	accountHandler := handlers.NewAccountHandler(clients.Account)
	transactionHandler := handlers.NewTransactionHandler(clients.Transaction)

	engine := router.NewRouter(validator, authHandler, systemHandler, accountHandler, transactionHandler)

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      engine,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("server starting", "address", cfg.HTTPServer.Address)
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err = srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "err", err)
	}

	log.Info("server stopped")
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
