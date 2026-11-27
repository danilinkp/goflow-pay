package tracing

import (
	"context"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func Init(ctx context.Context, service string) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}
	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(strings.TrimRight(endpoint, "/")+"/v1/traces"))
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(service)))
	if err != nil {
		return nil, err
	}
	provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return provider.Shutdown, nil
}

func TraceID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

type incomingCarrier struct{ metadata.MD }

func (c incomingCarrier) Get(key string) string {
	v := c.MD.Get(key)
	if len(v) > 0 {
		return v[0]
	}
	return ""
}
func (c incomingCarrier) Set(key, value string) { c.MD.Set(key, value) }
func (c incomingCarrier) Keys() []string {
	keys := make([]string, 0, len(c.MD))
	for k := range c.MD {
		keys = append(keys, k)
	}
	return keys
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			ctx = otel.GetTextMapPropagator().Extract(ctx, incomingCarrier{md})
		}
		ctx, span := otel.Tracer("goflow/grpc").Start(ctx, info.FullMethod)
		defer span.End()
		resp, err := handler(ctx, req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, status.Code(err).String())
		}
		return resp, err
	}
}

func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return UnaryClientInterceptorWithTimeout(3 * time.Second)
}

func UnaryClientInterceptorWithTimeout(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, span := otel.Tracer("goflow/grpc").Start(ctx, method)
		defer span.End()
		span.SetAttributes(attribute.String("server.address", cc.Target()))
		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		otel.GetTextMapPropagator().Inject(ctx, incomingCarrier{md})
		ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(ctx, md), timeout)
		defer cancel()
		err := invoker(ctx, method, req, reply, cc, opts...)
		name := method[strings.LastIndex(method, "/")+1:]
		if (strings.HasPrefix(name, "Get") || strings.HasPrefix(name, "List") || name == "IsTokenValid") &&
			(status.Code(err) == grpccodes.Unavailable || status.Code(err) == grpccodes.DeadlineExceeded) && ctx.Err() == nil {
			timer := time.NewTimer(50 * time.Millisecond)
			select {
			case <-timer.C:
				err = invoker(ctx, method, req, reply, cc, opts...)
			case <-ctx.Done():
				err = ctx.Err()
			}
			timer.Stop()
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, status.Code(err).String())
		}
		return err
	}
}
