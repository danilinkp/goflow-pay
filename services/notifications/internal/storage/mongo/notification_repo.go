package mongo

import (
	"context"
	"errors"
	"fmt"
	"notifications/internal/domain"
	"notifications/internal/domain/entities"
	"notifications/internal/storage/mongo/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type NotificationRepo struct {
	collection *mongo.Collection
}

func NewNotificationRepo(db *mongo.Database) *NotificationRepo {
	return &NotificationRepo{
		collection: db.Collection("notifications"),
	}
}

func (r *NotificationRepo) Save(ctx context.Context, notification *entities.Notification) error {
	op := "NotificationRepo.Save"
	notificationModel := models.ToNotificationModel(notification)

	_, err := r.collection.InsertOne(ctx, notificationModel)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, domain.ErrNotificationAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *NotificationRepo) GetById(ctx context.Context, id uuid.UUID) (*entities.Notification, error) {
	op := "NotificationRepo.GetById"
	var notificationModel models.NotificationModel
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&notificationModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrNotificationNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return notificationModel.ToDomain(), nil
}
