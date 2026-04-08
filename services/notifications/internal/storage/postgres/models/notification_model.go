package models

import (
	"notifications/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type NotificationModel struct {
	Id        uuid.UUID `db:"id"`
	UserId    uuid.UUID `db:"user_id"`
	Title     string    `db:"title"`
	Message   string    `db:"message"`
	SourceId  uuid.UUID `db:"source_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (m *NotificationModel) ToDomain() *entities.Notification {
	return entities.ReconstructNotification(m.Id, m.UserId, m.Title, m.Message, m.SourceId, m.CreatedAt, m.UpdatedAt)
}

func ToNotificationModel(in *entities.Notification) *NotificationModel {
	return &NotificationModel{
		Id:        in.ID(),
		UserId:    in.UserID(),
		Title:     in.Title(),
		Message:   in.Message(),
		SourceId:  in.SourceID(),
		CreatedAt: in.CreatedAt(),
		UpdatedAt: in.UpdatedAt(),
	}
}
