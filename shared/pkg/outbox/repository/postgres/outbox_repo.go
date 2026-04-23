package postgres

import (
	"context"
	"fmt"
	"shared/pkg/outbox"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/google/uuid"
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
		event.PublishedAt,
		event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *OutboxRepo) FetchPending(ctx context.Context, limit int) ([]*outbox.Event, error) {
	op := "OutboxRepo.FetchPending"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `
        SELECT id, aggregate_id, aggregate_type, event_type, topic, payload, failed_reason, attempts, created_at, published_at
        FROM outbox
        WHERE published_at IS NULL AND attempts < 5
        ORDER BY created_at ASC
        LIMIT $1
        FOR UPDATE SKIP LOCKED`

	rows, err := conn.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var events []*outbox.Event
	for rows.Next() {
		var e outbox.Event
		err := rows.Scan(
			&e.ID,
			&e.AggregateID,
			&e.AggregateType,
			&e.EventType,
			&e.Topic,
			&e.Payload,
			&e.FailedReason,
			&e.Attempts,
			&e.CreatedAt,
			&e.PublishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s (scan): %w", op, err)
		}
		events = append(events, &e)
	}

	return events, nil
}

func (r *OutboxRepo) MarkPublished(ctx context.Context, id uuid.UUID) error {
	op := "OutboxRepo.MarkPublished"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `UPDATE outbox SET published_at = NOW(), failed_reason = NULL WHERE id = $1`

	_, err := conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *OutboxRepo) MarkFailed(ctx context.Context, id uuid.UUID, reason string) error {
	op := "OutboxRepo.MarkFailed"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `
        UPDATE outbox 
        SET attempts = attempts + 1, 
            failed_reason = $2 
        WHERE id = $1`

	_, err := conn.Exec(ctx, query, id, reason)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
