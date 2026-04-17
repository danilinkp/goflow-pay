package kafka

import (
	"context"
	"fmt"

	"shared/pkg/outbox"

	"github.com/segmentio/kafka-go"
)

type Publisher struct {
	writer *kafka.Writer
}

func NewPublisher(brokers []string) *Publisher {
	return &Publisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		},
	}
}

func (p *Publisher) Publish(ctx context.Context, event *outbox.Event) error {
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: event.Topic,
		Key:   []byte(event.AggregateID.String()),
		Value: event.Payload,
		Headers: []kafka.Header{
			{
				Key:   "event_type",
				Value: []byte(event.EventType),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("kafka.Publish: %w", err)
	}

	return nil
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}
