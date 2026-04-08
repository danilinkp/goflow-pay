package repository

import (
	"context"
	"fmt"
	"shared/outbox"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxRepo struct {
	pool   *pgxpool.Pool
	getter *trmpgx.CtxGetter
}

func NewOutboxRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *OutboxRepo {
	return &OutboxRepo{
		pool:   pool,
		getter: c,
	}
}

func (r *OutboxRepo) Save(ctx context.Context, event *outbox.Event) error {
	op := "OutboxRepo.Save"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `INSERT INTO outbox (id, aggregate_id, aggregate_type, event_type, topic, payload, failed_reason, attempts, published_at, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := conn.Exec(ctx, query,
		event.ID,
		event.AggregateID,
		event.AggregateType,
		event.EventType,
		event.Topic,
		event.Payload,
		event.FailedReason,
		event.Attempts,
		event.CreatedAt,
		event.PublishedAt,
	)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
