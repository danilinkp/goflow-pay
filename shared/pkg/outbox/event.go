package outbox

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/propagation"
)

type EventType string

const (
	EventTransferCompleted       EventType = "notification.transfer.completed"
	EventTransferFailed          EventType = "notification.transfer.failed"
	EventBankDepositCompleted    EventType = "notification.deposit.completed"
	EventBankDepositFailed       EventType = "notification.deposit.failed"
	EventBankWithdrawalCompleted EventType = "notification.withdrawal.completed"
	EventBankWithdrawalFailed    EventType = "notification.withdrawal.failed"
	EventStatementGenerated      EventType = "notification.statement.generated"
)

func (e EventType) Topic() string {
	switch {
	case strings.HasPrefix(string(e), "notification."):
		return "notifications"
	default:
		return "events"
	}
}

type Event struct {
	ID            uuid.UUID  `db:"id" bson:"_id"`
	AggregateID   uuid.UUID  `db:"aggregate_id" bson:"aggregate_id"`
	AggregateType string     `db:"aggregate_type" bson:"aggregate_type"`
	EventType     EventType  `db:"event_type" bson:"event_type"`
	Topic         string     `db:"topic" bson:"topic"`
	Payload       []byte     `db:"payload" bson:"payload"`
	Traceparent   string     `db:"traceparent" bson:"traceparent"`
	Tracestate    string     `db:"tracestate" bson:"tracestate"`
	PublishedAt   *time.Time `db:"published_at" bson:"published_at"`
	FailedReason  *string    `db:"failed_reason" bson:"failed_reason"`
	Attempts      int        `db:"attempts" bson:"attempts"`
	CreatedAt     time.Time  `db:"created_at" bson:"created_at"`
	LockedUntil   *time.Time `db:"locked_until" bson:"locked_until"`
}

// CaptureTrace preserves the request trace across the outbox transaction boundary.
func (e *Event) CaptureTrace(ctx context.Context) {
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)
	e.Traceparent = carrier.Get("traceparent")
	e.Tracestate = carrier.Get("tracestate")
}

// RestoreTrace continues the originating request trace in the background worker.
func (e *Event) RestoreTrace(ctx context.Context) context.Context {
	if e.Traceparent == "" {
		return ctx
	}
	return propagation.TraceContext{}.Extract(ctx, propagation.MapCarrier{
		"traceparent": e.Traceparent,
		"tracestate":  e.Tracestate,
	})
}

type Outboxable interface {
	ToOutboxEvent() (*Event, error)
}

func WithType(base *Event, eventType EventType) *Event {
	base.ID = uuid.New()
	base.EventType = eventType
	base.Topic = eventType.Topic()
	return base
}
