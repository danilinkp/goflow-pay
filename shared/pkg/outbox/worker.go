package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type OutboxRepository interface {
	Save(ctx context.Context, event *Event) error
	FetchPending(ctx context.Context, limit int) ([]*Event, error)
	MarkPublished(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string) error
	EnsureIndexes(ctx context.Context) error
}

type Publisher interface {
	Publish(ctx context.Context, event *Event) error
}

type Worker struct {
	log       *slog.Logger
	repo      OutboxRepository
	publisher Publisher
	interval  time.Duration
	batchSize int
}

func NewWorker(log *slog.Logger, repo OutboxRepository, publisher Publisher, interval time.Duration, batchSize int) *Worker {
	return &Worker{
		repo:      repo,
		publisher: publisher,
		interval:  interval,
		batchSize: batchSize,
		log:       log,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	const op = "outbox.Worker.processBatch"

	log := w.log.With("op", op)

	events, err := w.repo.FetchPending(ctx, w.batchSize)
	if err != nil {
		log.Error("failed to fetch pending events", "error", err)
		return
	}

	if len(events) == 0 {
		return
	}

	log.Debug("fetched events for processing", "count", len(events))

	for _, event := range events {
		select {
		case <-ctx.Done():
			log.Info("context cancelled, stopping batch processing")
			return
		default:
		}

		eventLog := log.With(
			"event_id", event.ID,
			"event_type", event.EventType,
		)

		if err = w.publisher.Publish(ctx, event); err != nil {
			eventLog.Error("failed to publish event", "error", err)

			if markErr := w.repo.MarkFailed(ctx, event.ID, err.Error()); markErr != nil {
				eventLog.Error("failed to mark event as failed in db", "error", markErr)
			}
			continue
		}

		if markErr := w.repo.MarkPublished(ctx, event.ID); markErr != nil {
			eventLog.Error("failed to mark event as published in db", "error", markErr)
			continue
		}

		eventLog.Info("event successfully published and marked")
	}
}
