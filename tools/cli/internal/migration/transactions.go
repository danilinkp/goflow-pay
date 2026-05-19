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

func MigrateTransactions(ctx context.Context, postgresDSN, mongoDSN, mongoDBName string) error {
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

	fmt.Println("  transactions...")

	rows, err := pg.Query(ctx, `
		SELECT transaction_id, initiator_id, from_account_id, to_account_id,
		       amount, currency, idempotency_key, status, created_at, updated_at
		FROM transactions
	`)
	if err != nil {
		return fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var docs []interface{}
	for rows.Next() {
		var (
			transactionId  uuid.UUID
			initiatorId    uuid.UUID
			fromAccountId  uuid.UUID
			toAccountId    uuid.UUID
			amount         int64
			currency       string
			idempotencyKey string
			status         string
			createdAt      time.Time
			updatedAt      time.Time
		)
		if err = rows.Scan(&transactionId, &initiatorId, &fromAccountId, &toAccountId,
			&amount, &currency, &idempotencyKey, &status, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan transaction: %w", err)
		}
		docs = append(docs, bson.M{
			"_id":             transactionId,
			"initiator_id":    initiatorId,
			"from_account_id": fromAccountId,
			"to_account_id":   toAccountId,
			"amount":          amount,
			"currency":        currency,
			"idempotency_key": idempotencyKey,
			"status":          status,
			"created_at":      createdAt,
			"updated_at":      updatedAt,
		})
	}

	return bulkInsert(ctx, db.Collection("transactions"), docs)
}
