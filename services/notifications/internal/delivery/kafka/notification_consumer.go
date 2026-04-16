package notificationskafka

import (
	"context"
	"encoding/json"
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

func (c *NotificationConsumer) Read(ctx context.Context) error {
	defer c.reader.Close()

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		var event outbox.Event
		if err = json.Unmarshal(m.Value, &event); err != nil {
			c.logger.Error("invalid event format", slog.Any("payload", m.Value))
			c.reader.CommitMessages(ctx, m)
			continue
		}

		err = c.dispatch(ctx, event)
		if err != nil {
			c.logger.Error("failed to dispatch event",
				slog.String("event_id", event.ID.String()),
				slog.String("error", err.Error()),
			)
			time.Sleep(2 * time.Second)
			continue
		}

		if err = c.reader.CommitMessages(ctx, m); err != nil {
			return err
		}
	}
}

func (c *NotificationConsumer) dispatch(ctx context.Context, event outbox.Event) error {
	switch event.EventType {
	case outbox.EventTransferCompleted:
		var p TransferPayload
		err := json.Unmarshal(event.Payload, &p)
		if err != nil {
			return err
		}
		return c.service.NotifyTransferCompleted(ctx, p.InitiatorID, p.TransactionID, p.Amount, p.Currency)

	case outbox.EventTransferFailed:
		var p TransferPayload
		err := json.Unmarshal(event.Payload, &p)
		if err != nil {
			return err
		}
		return c.service.NotifyTransferFailed(ctx, p.InitiatorID, p.TransactionID, p.Amount, p.Currency)

	case outbox.EventBankDepositCompleted:
		var p BankOpPayload
		err := json.Unmarshal(event.Payload, &p)
		if err != nil {
			return err
		}
		return c.service.NotifyBankDepositCompleted(ctx, p.InitiatorID, p.AccountID, p.Amount, p.Currency)

	case outbox.EventBankDepositFailed:
		var p BankOpPayload
		err := json.Unmarshal(event.Payload, &p)
		if err != nil {
			return err
		}
		return c.service.NotifyBankDepositFailed(ctx, p.InitiatorID, p.AccountID, p.Amount, p.Currency)

	case outbox.EventBankWithdrawalCompleted:
		var p BankOpPayload
		err := json.Unmarshal(event.Payload, &p)
		if err != nil {
			return err
		}
		return c.service.NotifyBankWithdrawalCompleted(ctx, p.InitiatorID, p.AccountID, p.Amount, p.Currency)

	case outbox.EventBankWithdrawalFailed:
		var p BankOpPayload
		err := json.Unmarshal(event.Payload, &p)
		if err != nil {
			return err
		}
		return c.service.NotifyBankWithdrawalFailed(ctx, p.InitiatorID, p.AccountID, p.Amount, p.Currency)
	}
	return nil
}
