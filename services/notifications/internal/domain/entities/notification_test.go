package entities_test

import (
	"notifications/internal/domain/entities"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNotification_Success(t *testing.T) {
	userId := uuid.New()
	sourceId := uuid.New()

	n, err := entities.NewNotification(userId, "Title", "Message", sourceId)

	require.NoError(t, err)
	assert.Equal(t, userId, n.UserId())
	assert.Equal(t, "Title", n.Title())
	assert.Equal(t, "Message", n.Message())
	assert.Equal(t, sourceId, n.SourceId())
	assert.NotEqual(t, uuid.Nil, n.NotificationId())
}

func TestNewNotification_NilUserId(t *testing.T) {
	_, err := entities.NewNotification(uuid.Nil, "Title", "Message", uuid.New())

	assert.Error(t, err)
}

func TestNewNotification_EmptyTitle(t *testing.T) {
	_, err := entities.NewNotification(uuid.New(), "", "Message", uuid.New())

	assert.Error(t, err)
}

func TestNewNotification_EmptyMessage(t *testing.T) {
	_, err := entities.NewNotification(uuid.New(), "Title", "", uuid.New())

	assert.Error(t, err)
}

func TestNotification_UpdateTitle_Success(t *testing.T) {
	n, _ := entities.NewNotification(uuid.New(), "Old", "Message", uuid.New())

	err := n.UpdateTitle("New")

	require.NoError(t, err)
	assert.Equal(t, "New", n.Title())
}

func TestNotification_UpdateTitle_Empty(t *testing.T) {
	n, _ := entities.NewNotification(uuid.New(), "Old", "Message", uuid.New())

	err := n.UpdateTitle("")

	assert.Error(t, err)
	assert.Equal(t, "Old", n.Title())
}

func TestNotification_UpdateMessage_Success(t *testing.T) {
	n, _ := entities.NewNotification(uuid.New(), "Title", "Old", uuid.New())

	err := n.UpdateMessage("New")

	require.NoError(t, err)
	assert.Equal(t, "New", n.Message())
}

func TestNotification_UpdateMessage_Empty(t *testing.T) {
	n, _ := entities.NewNotification(uuid.New(), "Title", "Old", uuid.New())

	err := n.UpdateMessage("")

	assert.Error(t, err)
	assert.Equal(t, "Old", n.Message())
}
