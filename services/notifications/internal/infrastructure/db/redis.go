package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, connectionString string) (*redis.Client, error) {
	op := "db.NewRedisClient"

	opts, err := redis.ParseURL(connectionString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	client := redis.NewClient(opts)
	if err = client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return client, nil
}
