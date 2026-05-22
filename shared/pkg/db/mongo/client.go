package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func NewClient(ctx context.Context, uri string, timeout time.Duration) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo.NewClient: %w", err)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)
	for {
		pingErr := client.Ping(ctx, readpref.Primary())
		if pingErr == nil {
			return client, nil
		}
		if time.Now().After(deadline) {
			_ = client.Disconnect(ctx)
			return nil, fmt.Errorf("mongo.NewClient: ping: %w", pingErr)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
