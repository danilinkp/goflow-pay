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

func MigrateTransactions(ctx context.Context, postgresDSN, mongoDSN, mongoDBName string, batchSize int) error {
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

	tx, err := pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DECLARE cur CURSOR FOR
		SELECT transaction_id, initiator_id, from_account_id, to_account_id,
		       amount, currency, idempotency_key, status, created_at, updated_at
		FROM transactions`)
	if err != nil {
		return fmt.Errorf("declare cursor: %w", err)
	}

	collection := db.Collection("transactions")
	total := 0

	for {
		rows, err := tx.Query(ctx, fmt.Sprintf("FETCH %d FROM cur", batchSize))
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}

		var batch []interface{}
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
				rows.Close()
				return fmt.Errorf("scan: %w", err)
			}
			batch = append(batch, bson.M{
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
		rows.Close()

		if len(batch) == 0 {
			break
		}

		if err = flushBatch(ctx, collection, batch); err != nil {
			return err
		}
		total += len(batch)
		fmt.Printf("    inserted %d docs\n", total)
	}

	fmt.Printf("    total: %d\n", total)
	return nil
}
