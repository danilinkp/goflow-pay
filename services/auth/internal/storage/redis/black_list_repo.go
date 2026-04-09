package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type BlackListRepository struct {
	client *redis.Client
	prefix string
}

func NewBlackListRepository(client *redis.Client) *BlackListRepository {
	return &BlackListRepository{
		client: client,
		prefix: "blacklist:",
	}
}

func (r *BlackListRepository) Save(ctx context.Context, tokenID string, tokenTTL time.Duration) error {
	op := "BlackListRepository.Save"

	err := r.client.Set(ctx, r.prefix+tokenID, "1", tokenTTL).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *BlackListRepository) Exists(ctx context.Context, tokenID string) (bool, error) {
	op := "BlackListRepository.Exists"

	n, err := r.client.Exists(ctx, r.prefix+tokenID).Result()
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return n > 0, nil
}
