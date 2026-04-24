package outbox

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTransferCompleted       EventType = "notification.transfer.completed"
	EventTransferFailed          EventType = "notification.transfer.failed"
	EventBankDepositCompleted    EventType = "notification.deposit.completed"
	EventBankDepositFailed       EventType = "notification.deposit.failed"
	EventBankWithdrawalCompleted EventType = "notification.withdrawal.completed"
	EventBankWithdrawalFailed    EventType = "notification.withdrawal.failed"
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
	ID            uuid.UUID  `db:"id"`
	AggregateID   uuid.UUID  `db:"aggregate_id"`
	AggregateType string     `db:"aggregate_type"`
	EventType     EventType  `db:"event_type"`
	Topic         string     `db:"topic"`
	Payload       []byte     `db:"payload"`
	PublishedAt   *time.Time `db:"published_at"`
	FailedReason  *string    `db:"failed_reason"`
	Attempts      int        `db:"attempts"`
	CreatedAt     time.Time  `db:"created_at"`
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
