package appkafka

import (
	"context"
	"fmt"
	"log/slog"
	notificationskafka "notifications/internal/delivery/kafka"
	"notifications/internal/service"
)

type NotificationApp struct {
	log      *slog.Logger
	consumer *notificationskafka.NotificationConsumer
}

func NewNotificationApp(
	log *slog.Logger,
	brokers []string,
	topic string,
	service *service.NotificationService,
) *NotificationApp {
	consumer := notificationskafka.NewNotificationConsumer(brokers, topic, service, log)

	return &NotificationApp{
		log:      log,
		consumer: consumer,
	}
}

func (a *NotificationApp) Run(ctx context.Context) error {
	const op = "NotificationApp.Run"

	a.log.Info("starting notification consumer", slog.String("op", op))

	if err := a.consumer.Run(ctx); err != nil {
		return fmt.Errorf("NotificationApp.Run: %w", err)
	}
	return nil
}
