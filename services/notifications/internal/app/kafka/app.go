package appkafka

import (
	"context"
	"log/slog"
	notificationskafka "notifications/internal/delivery/kafka"
	"notifications/internal/services"
)

type NotificationApp struct {
	log      *slog.Logger
	consumer *notificationskafka.NotificationConsumer
}

func NewNotificationApp(
	log *slog.Logger,
	brokers []string,
	topic string,
	service *services.NotificationService,
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

	errChan := make(chan error, 1)

	go func() {
		if err := a.consumer.Read(ctx); err != nil {
			a.log.Error("consumer stopped with error", slog.String("err", err.Error()))
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		a.log.Info("stopping notification app by context")
		return nil
	case err := <-errChan:
		return err
	}
}
