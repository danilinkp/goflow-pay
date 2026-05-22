package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoAdapter struct {
	client *mongo.Client
}

func NewMongoAdapter(client *mongo.Client) *MongoAdapter {
	return &MongoAdapter{client: client}
}

func (a *MongoAdapter) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return a.client.UseSession(ctx, func(sessCtx context.Context) error {
		sess := mongo.SessionFromContext(sessCtx)

		if err := sess.StartTransaction(); err != nil {
			return err
		}

		if err := fn(sessCtx); err != nil {
			_ = sess.AbortTransaction(sessCtx)
			return err
		}

		return sess.CommitTransaction(sessCtx)
	})
}
