package main

import (
	"context"
	"errors"
	grpcClient "gateway/internal/clients/grpc"
	"gateway/internal/config"
	"gateway/internal/delivery/http/handlers"
	"gateway/internal/delivery/http/router"
	"net/http"
	"os"
	"os/signal"
	jwtValidator "shared/pkg/jwt"
	"shared/pkg/logger"
	"shared/pkg/logger/sl"
	"syscall"
	"time"
)

func main() {
	cfg := config.MustLoad()

	start := time.Now()
	log := logger.New(cfg.Env)

	log.Info("starting gateway service", "env", cfg.Env)

	publicKeyBytes, err := os.ReadFile(cfg.JWTPublicKeyPath)
	if err != nil {
		log.Error("failed to read JWT public key", "err", sl.Err(err))
	}
	validator, err := jwtValidator.NewValidator(publicKeyBytes)
	if err != nil {
		log.Error("failed to create validator", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		os.Exit(1)
	}

	clients, err := grpcClient.InitClients(
		cfg.GRPCClients.AuthAddress,
		cfg.GRPCClients.AccountAddress,
		cfg.GRPCClients.TransactionAddress,
	)
	if err != nil {
		log.Error("failed to init grpc clients", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
		os.Exit(1)
	}
	defer clients.Close()

	authHandler := handlers.NewAuthHandler(clients.Auth)
	systemHandler := handlers.NewSystemHandler(clients.Auth, cfg.BootstrapToken)
	accountHandler := handlers.NewAccountHandler(clients.Account)
	transactionHandler := handlers.NewTransactionHandler(clients.Transaction)

	engine := router.NewRouter(validator, authHandler, clients.Auth, systemHandler, accountHandler, transactionHandler)

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
			log.Error("server error", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err = srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", sl.ErrWithStack(err), sl.Duration(time.Since(start)))
	}

	log.Info("server stopped")
}
