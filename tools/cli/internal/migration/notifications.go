package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func MigrateNotifications(ctx context.Context, postgresDSN, mongoDSN, mongoDBName string) error {
	pg, err := pgxpool.New(ctx, postgresDSN)
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	defer pg.Close()

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(mongoDSN))
	if err != nil {
		return fmt.Errorf("mongo connect: %w", err)
	}
	defer mongoClient.Disconnect(ctx)

	db := mongoClient.Database(mongoDBName)

	fmt.Println("  notifications...")

	rows, err := pg.Query(ctx, `
		SELECT notification_id, user_id, title, message, source_id, created_at, updated_at
		FROM notifications
	`)
	if err != nil {
		return fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	var docs []interface{}
	for rows.Next() {
		var (
			notificationId uuid.UUID
			userId         uuid.UUID
			title          *string
			message        *string
			sourceId       uuid.UUID
			createdAt      time.Time
			updatedAt      time.Time
		)
		if err = rows.Scan(&notificationId, &userId, &title, &message, &sourceId, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan notification: %w", err)
		}
		docs = append(docs, bson.M{
			"_id":        notificationId,
			"user_id":    userId,
			"title":      title,
			"message":    message,
			"source_id":  sourceId,
			"created_at": createdAt,
			"updated_at": updatedAt,
		})
	}

	return bulkInsert(ctx, db.Collection("notifications"), docs)
}
