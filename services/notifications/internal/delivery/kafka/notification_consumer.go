package notificationskafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"notifications/internal/service"
	"shared/pkg/outbox"
	"shared/pkg/tracing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type NotificationConsumer struct {
	service *service.NotificationService
	reader  *kafka.Reader
	logger  *slog.Logger
}

func NewNotificationConsumer(brokers []string, topic string, service *service.NotificationService, logger *slog.Logger) *NotificationConsumer {
	c := kafka.ReaderConfig{
		Brokers:         brokers,
		Topic:           topic,
		MinBytes:        10e3,
		MaxBytes:        10e6,
		MaxWait:         1 * time.Second,
		ReadLagInterval: -1,
		GroupID:         "notification-consumer-group",
		StartOffset:     kafka.LastOffset,
	}
	return &NotificationConsumer{
		service: service,
		reader:  kafka.NewReader(c),
		logger:  logger,
	}
}

func (c *NotificationConsumer) Run(ctx context.Context) error {
	defer func() { _ = c.reader.Close() }()

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		eventType := extractEventType(m.Headers)
		carrier := propagation.MapCarrier{}
		for _, header := range m.Headers {
			carrier[header.Key] = string(header.Value)
		}
		eventCtx, span := otel.Tracer("goflow/kafka").Start(otel.GetTextMapPropagator().Extract(ctx, carrier), "consume "+string(eventType))
		eventID := carrier["event_id"]
		for _, header := range m.Headers {
			if header.Key == "event_id" {
				if id, parseErr := uuid.Parse(string(header.Value)); parseErr == nil {
					eventCtx = service.WithEventID(eventCtx, id)
				}
			}
		}

		for attempt := 1; attempt <= 3; attempt++ {
			err = c.dispatch(eventCtx, eventType, m.Value)
			if err == nil {
				break
			}
			c.logger.Error("failed to dispatch event", slog.String("event_id", eventID), slog.String("event_type", string(eventType)), slog.String("trace_id", tracing.TraceID(eventCtx)), slog.Int("attempt", attempt), slog.String("error", err.Error()))
			if attempt < 3 {
				select {
				case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
				case <-ctx.Done():
					return nil
				}
			}
		}
		if err != nil {
			span.RecordError(err)
			span.End()
			return fmt.Errorf("event processing failed after 3 attempts: %w", err)
		}
		c.logger.Info("event processed", slog.String("event_id", eventID), slog.String("event_type", string(eventType)), slog.String("trace_id", tracing.TraceID(eventCtx)))
		span.End()

		if err = c.reader.CommitMessages(ctx, m); err != nil {
			return fmt.Errorf("commit message: %w", err)
		}
	}
}

func extractEventType(headers []kafka.Header) outbox.EventType {
	for _, h := range headers {
		if h.Key == "event_type" {
			return outbox.EventType(h.Value)
		}
	}
	return ""
}

func (c *NotificationConsumer) dispatch(ctx context.Context, eventType outbox.EventType, payload []byte) error {
	switch eventType {
	case outbox.EventTransferCompleted:
		var p TransferPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal TransferPayload: %w", err)
		}
		return c.service.NotifyTransferCompleted(ctx, p.InitiatorID, p.TransactionID, p.Amount, p.Currency)

	case outbox.EventTransferFailed:
		var p TransferPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal TransferPayload: %w", err)
		}
		return c.service.NotifyTransferFailed(ctx, p.InitiatorID, p.TransactionID, p.Amount, p.Currency)

	case outbox.EventBankDepositCompleted:
		var p BankOpPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal BankOpPayload: %w", err)
		}
		return c.service.NotifyBankDepositCompleted(ctx, p.InitiatorID, p.AccountID, p.Amount, p.Currency)

	case outbox.EventBankDepositFailed:
		var p BankOpPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal BankOpPayload: %w", err)
		}
		return c.service.NotifyBankDepositFailed(ctx, p.InitiatorID, p.AccountID, p.Amount, p.Currency)

	case outbox.EventBankWithdrawalCompleted:
		var p BankOpPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal BankOpPayload: %w", err)
		}
		return c.service.NotifyBankWithdrawalCompleted(ctx, p.InitiatorID, p.AccountID, p.Amount, p.Currency)

	case outbox.EventBankWithdrawalFailed:
		var p BankOpPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal BankOpPayload: %w", err)
		}
		return c.service.NotifyBankWithdrawalFailed(ctx, p.InitiatorID, p.AccountID, p.Amount, p.Currency)

	case outbox.EventStatementGenerated:
		var p StatementPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal StatementPayload: %w", err)
		}

		var domainEntries []service.StatementEntry
		if len(p.Entries) > 0 {
			if err := json.Unmarshal(p.Entries, &domainEntries); err != nil {
				return fmt.Errorf("unmarshal StatementEntries: %w", err)
			}
		}

		return c.service.NotifyStatementGenerated(ctx, p.InitiatorID, p.AccountID, p.PeriodFrom, p.PeriodTo, p.OpeningBalance,
			p.ClosingBalance, p.TotalDebit, p.TotalCredit, p.Currency, domainEntries)

	default:
		c.logger.Warn("unknown event type, skipping", slog.String("event_type", string(eventType)), slog.String("trace_id", tracing.TraceID(ctx)))
		return nil
	}
}
