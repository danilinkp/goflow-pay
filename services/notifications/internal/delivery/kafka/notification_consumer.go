package notificationskafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"notifications/internal/services"
	"shared/pkg/outbox"
	"time"

	"github.com/segmentio/kafka-go"
)

type NotificationConsumer struct {
	service *services.NotificationService
	reader  *kafka.Reader
	logger  *slog.Logger
}

func NewNotificationConsumer(brokers []string, topic string, service *services.NotificationService, logger *slog.Logger) *NotificationConsumer {
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
	defer c.reader.Close()

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		eventType := extractEventType(m.Headers)

		if err = c.dispatch(ctx, eventType, m.Value); err != nil {
			c.logger.Error("failed to dispatch event",
				slog.String("event_type", string(eventType)),
				slog.String("error", err.Error()),
			)
			continue
		}

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

	default:
		c.logger.Warn("unknown event type, skipping", slog.String("event_type", string(eventType)))
		return nil
	}
}
