package mongo

import (
	"accounts/internal/domain"
	"accounts/internal/domain/entities"
	"accounts/internal/storage/mongo/models"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type AccountOperationRepo struct {
	collection *mongo.Collection
}

func NewAccountOperationRepo(db *mongo.Database) *AccountOperationRepo {
	return &AccountOperationRepo{
		collection: db.Collection("accounts"),
	}
}

func (r *AccountOperationRepo) EnsureIndexes(ctx context.Context) error {
	op := "AccountOperationRepo.EnsureIndexes"

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "transaction_id", Value: 1}},
			Options: options.Index().SetName("idx_transaction_id"),
		},
		{
			Keys:    bson.D{{Key: "account_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_account_id_created_at"),
		},
		{
			Keys:    bson.D{{Key: "account_id", Value: 1}, {Key: "operation_status", Value: 1}},
			Options: options.Index().SetName("idx_account_id_status"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *AccountOperationRepo) Save(ctx context.Context, accountOp *entities.AccountOperation) error {
	op := "AccountOperationRepo.Save"

	accOpModel := models.ToAccountOperationModel(accountOp)

	_, err := r.collection.InsertOne(ctx, accOpModel)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, domain.ErrAccountOperationAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *AccountOperationRepo) GetByTransactionIdAndType(ctx context.Context, transactionId uuid.UUID, operationType string) (*entities.AccountOperation, error) {
	op := "AccountOperationRepo.GetByTransactionIdAndType"

	filter := bson.M{"transaction_id": transactionId, "operation_type": operationType}
	var accOpModel models.AccountOperationModel
	err := r.collection.FindOne(ctx, filter).Decode(&accOpModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrAccountOperationNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return accOpModel.ToDomain(), nil
}

func (r *AccountOperationRepo) GetByTransactionId(ctx context.Context, transactionId uuid.UUID) ([]*entities.AccountOperation, error) {
	op := "AccountOperationRepo.GetByTransactionId"

	filter := bson.M{"transaction_id": transactionId}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var ops []models.AccountOperationModel
	if err = cursor.All(ctx, &ops); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	res := make([]*entities.AccountOperation, len(ops))
	for i, row := range ops {
		res[i] = row.ToDomain()
	}

	return res, nil
}

func (r *AccountOperationRepo) UpdateStatus(ctx context.Context, operation uuid.UUID, status string) error {
	op := "AccountOperationRepo.UpdateStatus"

	now := time.Now().UTC()
	filter := bson.M{"_id": operation}
	update := bson.M{"$set": bson.M{"operation_status": status, "updated_at": now}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("%s: %w", op, domain.ErrAccountOperationNotFound)
	}

	return nil
}

func (r *AccountOperationRepo) ConfirmAllByTransactionId(ctx context.Context, transactionId uuid.UUID) error {
	op := "AccountOperationRepo.ConfirmAllByTransactionId"

	filter := bson.M{"transaction_id": transactionId}
	update := bson.M{"$set": bson.M{"operation_status": entities.SuccessStatus, "updated_at": time.Now().UTC()}}
	result, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("%s: %w", op, domain.ErrAccountOperationNotFound)
	}
	return nil
}

func (r *AccountOperationRepo) HasPendingByAccountId(ctx context.Context, accountId uuid.UUID) (bool, error) {
	op := "AccountOperationRepo.HasPendingByAccountId"

	filter := bson.M{"account_id": accountId, "operation_status": entities.PendingStatus}

	opts := options.Find().
		SetLimit(1).
		SetProjection(bson.M{"_id": 1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	return cursor.Next(ctx), nil
}

func (r *AccountOperationRepo) GetByAccountIdAndPeriod(ctx context.Context, accountId uuid.UUID, from, to time.Time) ([]*entities.AccountOperation, error) {
	op := "AccountOperationRepo.GetByAccountIdAndPeriod"

	toPlusOneDay := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location()).AddDate(0, 0, 1)

	filter := bson.M{
		"account_id": accountId,
		"created_at": bson.M{
			"$gte": from,
			"$lt":  toPlusOneDay,
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var ops []models.AccountOperationModel
	if err = cursor.All(ctx, &ops); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	res := make([]*entities.AccountOperation, len(ops))
	for i, row := range ops {
		res[i] = row.ToDomain()
	}

	return res, nil
}
