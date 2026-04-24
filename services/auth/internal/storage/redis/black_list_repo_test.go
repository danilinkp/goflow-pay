package redis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"auth/internal/storage/redis"
	db "shared/pkg/db/redis"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestBlackListRepository_Integration(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := rediscontainer.Run(ctx, "redis:8-alpine")
	require.NoError(t, err)
	defer func() {
		_ = redisContainer.Terminate(ctx)
	}()

	host, err := redisContainer.Host(ctx)
	require.NoError(t, err)

	port, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	addr := fmt.Sprintf("%s:%s", host, port.Port())

	client, err := db.NewRedisClient(ctx, addr, "", 0, time.Second, time.Second)
	require.NoError(t, err)

	repo := redis.NewBlackListRepository(client)

	t.Run("Save and Exists", func(t *testing.T) {
		tokenID := uuid.New().String()
		ttl := 1 * time.Hour

		exists, err := repo.Exists(ctx, tokenID)
		assert.NoError(t, err)
		assert.False(t, exists)

		err = repo.Save(ctx, tokenID, ttl)
		assert.NoError(t, err)

		exists, err = repo.Exists(ctx, tokenID)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Token Expired (TTL)", func(t *testing.T) {
		tokenID := uuid.New().String()
		shortTTL := 1 * time.Second

		err = repo.Save(ctx, tokenID, shortTTL)
		assert.NoError(t, err)

		time.Sleep(1100 * time.Millisecond)

		exists, err := repo.Exists(ctx, tokenID)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}
