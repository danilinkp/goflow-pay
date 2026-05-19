package models

import (
	"notifications/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type NotificationModel struct {
	NotificationId uuid.UUID `bson:"_id"`
	UserId         uuid.UUID `bson:"user_id"`
	Title          string    `bson:"title"`
	Message        string    `bson:"message"`
	SourceId       uuid.UUID `bson:"source_id"`
	CreatedAt      time.Time `bson:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at"`
}

func (m *NotificationModel) ToDomain() *entities.Notification {
	return entities.ReconstructNotification(m.NotificationId, m.UserId, m.Title, m.Message, m.SourceId, m.CreatedAt, m.UpdatedAt)
}

func ToNotificationModel(in *entities.Notification) *NotificationModel {
	return &NotificationModel{
		NotificationId: in.NotificationId(),
		UserId:         in.UserId(),
		Title:          in.Title(),
		Message:        in.Message(),
		SourceId:       in.SourceId(),
		CreatedAt:      in.CreatedAt(),
		UpdatedAt:      in.UpdatedAt(),
	}
}
