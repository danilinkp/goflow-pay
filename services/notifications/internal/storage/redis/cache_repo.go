package redis

import (
	"context"
	"errors"
	"fmt"
	"notifications/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CacheRepo struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCacheRepo(pool *redis.Client, ttl time.Duration) *CacheRepo {
	return &CacheRepo{
		client: pool,
	}
}

func (r *CacheRepo) GetCompanyId(ctx context.Context, accountId uuid.UUID) (uuid.UUID, error) {
	op := "CacheRepo.GetCompanyId"

	val, err := r.client.Get(ctx, "acc_comp:"+accountId.String()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, fmt.Errorf("%s: %w", op, domain.ErrCachedCompanyNotFound)
		}
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}
	companyId, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return companyId, nil
}

func (r *CacheRepo) SetCompanyId(ctx context.Context, accountId uuid.UUID, companyId uuid.UUID) error {
	op := "CacheRepo.SetCompanyId"

	if companyId == uuid.Nil {
		return nil
	}

	err := r.client.Set(ctx, "acc_comp:"+accountId.String(), companyId.String(), r.ttl).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
