package metrics

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	grpcHandled = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_server_handled_total",
		Help: "Total number of gRPC calls completed on the server, by method and status code.",
	}, []string{"method", "code"})

	grpcDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "grpc_server_handling_seconds",
		Help:    "Duration of gRPC calls handled by the server.",
		Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
	}, []string{"method"})
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		grpcDuration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())
		grpcHandled.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()
		return resp, err
	}
}

func Serve(log *slog.Logger, addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		log.Info("metrics server started", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server stopped", slog.String("error", err.Error()))
		}
	}()
}
