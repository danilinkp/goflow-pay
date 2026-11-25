package outbox

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestEventPreservesTraceAcrossOutbox(t *testing.T) {
	traceID := trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled,
	}))
	event := &Event{}
	event.CaptureTrace(ctx)
	if event.Traceparent == "" {
		t.Fatal("traceparent was not captured")
	}
	restored := event.RestoreTrace(context.Background())
	if got := trace.SpanContextFromContext(restored).TraceID(); got != traceID {
		t.Fatalf("restored trace ID = %s, want %s", got, traceID)
	}
}
