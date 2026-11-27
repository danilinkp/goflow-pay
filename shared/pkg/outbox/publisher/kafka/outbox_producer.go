package kafka

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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
	ctx = event.RestoreTrace(ctx)
	ctx, span := otel.Tracer("goflow/kafka").Start(ctx, "publish "+string(event.EventType))
	defer span.End()
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	headers := []kafka.Header{
		{Key: "event_type", Value: []byte(event.EventType)},
		{Key: "event_id", Value: []byte(event.ID.String())},
	}
	for key, value := range carrier {
		headers = append(headers, kafka.Header{Key: key, Value: []byte(value)})
	}
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic:   event.Topic,
		Key:     []byte(event.AggregateID.String()),
		Value:   event.Payload,
		Headers: headers,
	})

	if err != nil {
		return fmt.Errorf("kafka.Publish: %w", err)
	}

	return nil
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}
