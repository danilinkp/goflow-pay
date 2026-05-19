package mongo

import (
	"context"
	"errors"
	"fmt"
	"shared/pkg/outbox"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type OutboxRepo struct {
	collection *mongo.Collection
}

func NewOutboxRepo(db *mongo.Database) *OutboxRepo {
	return &OutboxRepo{
		collection: db.Collection("outbox"),
	}
}

func (r *OutboxRepo) EnsureIndexes(ctx context.Context) error {
	op := "OutboxRepo.EnsureIndexes"

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{"published_at", 1},
				{"attempts", 1},
				{"created_at", 1},
			},
			Options: options.Index().SetName("idx_outbox_pending"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *OutboxRepo) Save(ctx context.Context, event *outbox.Event) error {
	op := "OutboxRepo.Save"

	_, err := r.collection.InsertOne(ctx, event)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *OutboxRepo) FetchPending(ctx context.Context, limit int) ([]*outbox.Event, error) {
	op := "OutboxRepo.FetchPending"

	filter := bson.M{
		"published_at": nil,
		"attempts":     bson.M{"$lt": 5},
		"$or": bson.A{
			bson.M{"locked_until": nil},
			bson.M{"locked_until": bson.M{"$lt": time.Now().UTC()}},
		},
	}

	update := bson.M{
		"$set": bson.M{
			"locked_until": time.Now().Add(30 * time.Second),
		},
	}

	var events []*outbox.Event

	for i := 0; i < limit; i++ {
		opts := options.FindOneAndUpdate().
			SetSort(bson.D{{"created_at", 1}}).
			SetReturnDocument(options.After)

		var event outbox.Event
		err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&event)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				break
			}
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		events = append(events, &event)
	}

	return events, nil
}

func (r *OutboxRepo) MarkPublished(ctx context.Context, id uuid.UUID) error {
	op := "OutboxRepo.MarkPublished"

	now := time.Now()
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"published_at": &now,
			"locked_until": nil,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("%s: event not found", op)
	}

	return nil
}

func (r *OutboxRepo) MarkFailed(ctx context.Context, id uuid.UUID, reason string) error {
	op := "OutboxRepo.MarkFailed"

	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"failed_reason": &reason,
			"locked_until":  nil,
		},
		"$inc": bson.M{
			"attempts": 1,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("%s: event not found", op)
	}

	return nil
}
