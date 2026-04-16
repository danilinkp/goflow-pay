package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, addr, password string, db int, readTimeOut, writeTimeout time.Duration) (*redis.Client, error) {
	op := "db.NewRedisClient"

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  readTimeOut,
		WriteTimeout: writeTimeout,
		PoolSize:     10,
		MinIdleConns: 3,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return client, nil
}
