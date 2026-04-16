package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OutboxRepository interface {
	FetchPending(ctx context.Context, limit int) ([]*Event, error)
	MarkPublished(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string) error
}

type Publisher interface {
	Publish(ctx context.Context, event *Event) error
}

type Worker struct {
	repo       OutboxRepository
	publisher  Publisher
	interval   time.Duration
	batchSize  int
	maxRetries int
}

func NewWorker(repo OutboxRepository, publisher Publisher, interval time.Duration, batchSize int, maxRetries int) *Worker {
	return &Worker{
		repo:       repo,
		publisher:  publisher,
		interval:   interval,
		batchSize:  batchSize,
		maxRetries: maxRetries,
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
	events, err := w.repo.FetchPending(ctx, w.batchSize)
	if err != nil {
		return
	}
	for _, event := range events {
		if event.Attempts >= w.maxRetries {
			_ = w.repo.MarkFailed(ctx, event.ID, "max retries exceeded")
			continue
		}
		if err = w.publisher.Publish(ctx, event); err != nil {
			_ = w.repo.MarkFailed(ctx, event.ID, err.Error())
			continue
		}
		_ = w.repo.MarkPublished(ctx, event.ID)
	}
}
