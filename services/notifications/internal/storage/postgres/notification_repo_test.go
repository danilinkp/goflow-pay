package postgres_test

import (
	"context"
	"errors"
	"notifications/internal/domain"
	"notifications/internal/domain/entities"
	"notifications/internal/storage/postgres"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newNotificationRepo() *postgres.NotificationRepo {
	return postgres.NewNotificationRepo(testPool, testGetter)
}

func TestNotificationRepo_Save_And_GetById(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newNotificationRepo()

	userId := uuid.New()
	sourceId := uuid.New()

	notification, err := entities.NewNotification(
		userId,
		"Оплата прошла",
		"Ваш платеж на сумму 1000 RSD успешно обработан",
		sourceId,
	)
	require.NoError(t, err)

	err = repo.Save(ctx, notification)
	require.NoError(t, err)

	got, err := repo.GetById(ctx, notification.NotificationId())
	require.NoError(t, err)

	assert.Equal(t, notification.NotificationId(), got.NotificationId())
	assert.Equal(t, "Оплата прошла", got.Title())
	assert.Equal(t, sourceId, got.SourceId())
}

func TestNotificationRepo_GetById_NotFound(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newNotificationRepo()

	got, err := repo.GetById(ctx, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, domain.ErrNotificationNotFound))
}
