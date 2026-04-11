package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	op := "db.NewPostgresPool"

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("%s (parse): %w", op, err)
	}

	config.ConnConfig.ConnectTimeout = 3 * time.Second

	var pool *pgxpool.Pool

	deadline := time.Now().Add(10 * time.Second)

	for {
		pool, err = pgxpool.NewWithConfig(ctx, config)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err = pool.Ping(pingCtx)
			cancel()

			if err == nil {
				return pool, nil
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%s: database is unavailable after 10s: %w", op, err)
		}

		time.Sleep(1 * time.Second)
	}
}
