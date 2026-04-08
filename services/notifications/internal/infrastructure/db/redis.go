package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, addr string, password string, db int) (*redis.Client, error) {
	op := "db.NewRedisClient"

	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return client, nil
}
