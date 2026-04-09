package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	notificationId uuid.UUID
	userId         uuid.UUID
	title          string
	message        string
	sourceId       uuid.UUID
	createdAt      time.Time
	updatedAt      time.Time
}

func NewNotification(userId uuid.UUID, title, message string, sourceId uuid.UUID) (*Notification, error) {
	op := "create notification"
	if userId == uuid.Nil {
		return nil, fmt.Errorf("%s: user id is required", op)
	}
	if title == "" || message == "" {
		return nil, fmt.Errorf("%s: title and message cannot be blank", op)
	}

	now := time.Now().UTC()
	return &Notification{
		notificationId: uuid.New(),
		userId:         userId,
		title:          title,
		message:        message,
		sourceId:       sourceId,
		createdAt:      now,
	}, nil
}

func ReconstructNotification(id, userId uuid.UUID, title, message string, sourceId uuid.UUID, createdAt, updatedAt time.Time) *Notification {
	return &Notification{
		notificationId: id,
		userId:         userId,
		title:          title,
		message:        message,
		sourceId:       sourceId,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

func (n *Notification) NotificationId() uuid.UUID { return n.notificationId }
func (n *Notification) UserId() uuid.UUID         { return n.userId }
func (n *Notification) Title() string             { return n.title }
func (n *Notification) Message() string           { return n.message }
func (n *Notification) SourceId() uuid.UUID       { return n.sourceId }
func (n *Notification) CreatedAt() time.Time      { return n.createdAt }
func (n *Notification) UpdatedAt() time.Time      { return n.updatedAt }

func (n *Notification) UpdateTitle(newTitle string) error {
	if newTitle == "" {
		return fmt.Errorf("%s: title cannot be blank", "update notification title")
	}
	n.title = newTitle
	n.updatedAt = time.Now().UTC()
	return nil
}

func (n *Notification) UpdateMessage(newMessage string) error {
	if newMessage == "" {
		return fmt.Errorf("%s: message cannot be blank", "update notification message")
	}
	n.message = newMessage
	n.updatedAt = time.Now().UTC()
	return nil
}
