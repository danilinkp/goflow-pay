package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"
	"transactions/internal/domain"
	"transactions/internal/domain/entities"
	"transactions/internal/storage/mongo/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type TransactionRepo struct {
	collection *mongo.Collection
}

func NewTransactionRepo(db *mongo.Database) *TransactionRepo {
	return &TransactionRepo{
		collection: db.Collection("transactions"),
	}
}

func (r *TransactionRepo) EnsureIndexes(ctx context.Context) error {
	op := "TransactionRepo.EnsureIndexes"

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"idempotency_key", 1}},
			Options: options.Index().SetName("idx_idempotency_key").SetUnique(true),
		},
		{
			Keys:    bson.D{{"from_account_id", 1}, {"created_at", -1}},
			Options: options.Index().SetName("idx_from_account_id_created_at"),
		},
		{
			Keys:    bson.D{{"to_account_id", 1}, {"created_at", -1}},
			Options: options.Index().SetName("idx_to_account_id_created_at"),
		},
		{
			Keys:    bson.D{{"status", 1}, {"updated_at", 1}},
			Options: options.Index().SetName("idx_status_updated_at"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *TransactionRepo) Save(ctx context.Context, transaction *entities.Transaction) error {
	op := "TransactionRepo.Save"

	txModel := models.ToTransactionModel(transaction)
	_, err := r.collection.InsertOne(ctx, txModel)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, domain.ErrTransactionAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *TransactionRepo) GetById(ctx context.Context, id uuid.UUID) (*entities.Transaction, error) {
	op := "TransactionRepo.GetById"

	var txModel models.TransactionModel
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&txModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrTransactionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return txModel.ToDomain(), nil
}

func (r *TransactionRepo) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*entities.Transaction, error) {
	op := "TransactionRepo.GetByIdempotencyKey"

	var txModel models.TransactionModel
	filter := bson.M{"idempotency_key": idempotencyKey}
	err := r.collection.FindOne(ctx, filter).Decode(&txModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrTransactionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return txModel.ToDomain(), nil
}

func (r *TransactionRepo) GetByAccountId(ctx context.Context, accountId uuid.UUID) ([]*entities.Transaction, error) {
	op := "TransactionRepo.GetByAccountId"

	filter := bson.M{
		"$or": bson.A{
			bson.M{"from_account_id": accountId},
			bson.M{"to_account_id": accountId},
		},
	}

	opts := options.Find().SetSort(bson.D{{"created_at", -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var txModels []models.TransactionModel
	if err = cursor.All(ctx, &txModels); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	res := make([]*entities.Transaction, len(txModels))
	for i, tx := range txModels {
		res[i] = tx.ToDomain()
	}

	return res, nil
}

func (r *TransactionRepo) GetStale(ctx context.Context, olderThan time.Duration, statuses []string) ([]*entities.Transaction, error) {
	op := "TransactionRepo.GetStale"

	threshold := time.Now().Add(-olderThan)

	filter := bson.M{
		"status":     bson.M{"$in": statuses},
		"updated_at": bson.M{"$lt": threshold},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var txModels []models.TransactionModel
	if err = cursor.All(ctx, &txModels); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	res := make([]*entities.Transaction, len(txModels))
	for i, tx := range txModels {
		res[i] = tx.ToDomain()
	}

	return res, nil
}

func (r *TransactionRepo) UpdateStatus(ctx context.Context, txId uuid.UUID, status string) error {
	op := "TransactionRepo.UpdateStatus"

	filter := bson.M{"_id": txId}
	update := bson.M{"$set": bson.M{"status": status, "updated_at": time.Now().UTC()}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("%s: %w", op, domain.ErrTransactionNotFound)
	}

	return nil
}
