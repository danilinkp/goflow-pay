package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string, connTimeout, maxRetriesTime time.Duration) (*pgxpool.Pool, error) {
	op := "db.NewPostgresPool"

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%s (parse): %w", op, err)
	}

	poolConfig.ConnConfig.ConnectTimeout = connTimeout

	var pool *pgxpool.Pool

	retryCtx, cancel := context.WithTimeout(ctx, maxRetriesTime)
	defer cancel()

	for {
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err == nil {
			pingCtx, pingCancel := context.WithTimeout(retryCtx, connTimeout)
			err = pool.Ping(pingCtx)
			pingCancel()

			if err == nil {
				return pool, nil
			}
			pool.Close()
		}

		select {
		case <-retryCtx.Done():
			return nil, fmt.Errorf("%s: database unavailable after %s: %w", op, maxRetriesTime, err)
		case <-time.After(1 * time.Second):
			continue
		}
	}
}
