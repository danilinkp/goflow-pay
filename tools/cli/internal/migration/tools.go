package migration

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func bulkInsert(ctx context.Context, collection *mongo.Collection, docs []interface{}) error {
	if len(docs) == 0 {
		fmt.Println("    no data, skipping")
		return nil
	}

	opts := options.InsertMany().SetOrdered(false)
	_, err := collection.InsertMany(ctx, docs, opts)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			fmt.Printf("    skipped duplicates\n")
			return nil
		}
		return fmt.Errorf("insert: %w", err)
	}

	fmt.Printf("    inserted %d documents\n", len(docs))
	return nil
}

func flushBatch(ctx context.Context, collection *mongo.Collection, batch []interface{}) error {
	if len(batch) == 0 {
		return nil
	}
	opts := options.InsertMany().SetOrdered(false)
	_, err := collection.InsertMany(ctx, batch, opts)
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("insert batch: %w", err)
	}
	return nil
}
