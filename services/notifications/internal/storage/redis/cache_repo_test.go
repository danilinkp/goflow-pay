package redis_test

import (
	"context"
	"notifications/internal/infrastructure/db"
	"notifications/internal/storage/redis"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestCacheRepo_Integration(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := rediscontainer.Run(ctx,
		"redis:8-alpine",
	)
	require.NoError(t, err)
	defer func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	connectionString, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := db.NewRedisClient(ctx, connectionString)
	require.NoError(t, err)

	ttl := 5 * time.Minute
	repo := redis.NewCacheRepo(client, ttl)

	accountId := uuid.New()
	companyId := uuid.New()

	t.Run("Success Set and Get", func(t *testing.T) {
		err = repo.SetCompanyId(ctx, accountId, companyId)
		assert.NoError(t, err)

		got, err := repo.GetCompanyId(ctx, accountId)
		assert.NoError(t, err)
		assert.Equal(t, companyId, got)
	})

	t.Run("Get Non-Existent Key", func(t *testing.T) {
		got, err := repo.GetCompanyId(ctx, uuid.New())
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, got)
		assert.Contains(t, err.Error(), "cached company not found")
	})

	t.Run("Set Nil CompanyId Should Skip", func(t *testing.T) {
		newAccId := uuid.New()
		err := repo.SetCompanyId(ctx, newAccId, uuid.Nil)
		assert.NoError(t, err)

		got, err := repo.GetCompanyId(ctx, newAccId)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, got)
	})
}
