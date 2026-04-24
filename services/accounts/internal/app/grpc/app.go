package grpc

import (
	accountsgrpc "accounts/internal/delievery/grpc"
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AccountsApp struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

func NewAccountsApp(log *slog.Logger, accountsService accountsgrpc.AccountProvider, port int) *AccountsApp {
	loggingOpts := []logging.Option{
		logging.WithLogOnEvents(
			//logging.StartCall, logging.FinishCall,
			logging.PayloadReceived, logging.PayloadSent,
		),
	}

	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p interface{}) (err error) {
			log.Error("Recovered from panic", slog.Any("panic", p))

			return status.Errorf(codes.Internal, "internal error")
		}),
	}

	gRPCServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		recovery.UnaryServerInterceptor(recoveryOpts...),
		logging.UnaryServerInterceptor(InterceptorLogger(log), loggingOpts...),
		AuthInterceptor(),
	))

	accountsgrpc.Register(gRPCServer, accountsService)

	return &AccountsApp{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

func InterceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is missing")
		}
		userIDs := md.Get("x-user-id")
		if len(userIDs) == 0 {
			return nil, status.Error(codes.Unauthenticated, "x-user-id header is missing")
		}

		uid, err := uuid.Parse(userIDs[0])
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user id format")
		}

		newCtx := context.WithValue(ctx, accountsgrpc.UserIDKey, uid)
		return handler(newCtx, req)
	}
}

func (a *AccountsApp) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *AccountsApp) Run() error {
	const op = "grpcapp.Run"

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("grpc server started", slog.String("addr", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *AccountsApp) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).Info("stopping gRPC server", slog.Int("port", a.port))

	a.gRPCServer.GracefulStop()
}
