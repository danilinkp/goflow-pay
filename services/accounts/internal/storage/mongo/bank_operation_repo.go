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
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type BankOperationRepo struct {
	collection *mongo.Collection
}

func NewBankOperationRepo(db *mongo.Database) *BankOperationRepo {
	return &BankOperationRepo{
		collection: db.Collection("bank_operations"),
	}
}

func (r *BankOperationRepo) EnsureIndexes(ctx context.Context) error {
	op := "BankOperationRepo.EnsureIndexes"

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "idempotency_key", Value: 1}},
			Options: options.Index().SetName("idx_idempotency_key").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "account_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_account_id_created_at"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *BankOperationRepo) Save(ctx context.Context, operation *entities.BankOperation) error {
	op := "BankOperationRepo.Save"

	bankOpModel := models.ToBankOperationModel(operation)
	_, err := r.collection.InsertOne(ctx, bankOpModel)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, domain.ErrBankOperationAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *BankOperationRepo) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*entities.BankOperation, error) {
	op := "BankOperationRepo.GetByIdempotencyKey"

	filter := bson.M{"idempotency_key": idempotencyKey}
	var bankOpModel models.BankOperationModel
	err := r.collection.FindOne(ctx, filter).Decode(&bankOpModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrBankOperationNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return bankOpModel.ToDomain(), nil
}

func (r *BankOperationRepo) UpdateStatusAndExternalID(ctx context.Context, operationId uuid.UUID, status string, externalID string) error {
	op := "BankOperationRepo.UpdateStatusAndExternalID"

	filter := bson.M{"_id": operationId}
	update := bson.M{"$set": bson.M{"operation_status": status, "external_id": externalID}}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("%s, %w", op, domain.ErrBankOperationNotFound)
	}
	return nil
}

func (r *BankOperationRepo) GetByAccountIdAndPeriod(ctx context.Context, accountId uuid.UUID, from, to time.Time) ([]*entities.BankOperation, error) {
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

	var ops []models.BankOperationModel
	if err = cursor.All(ctx, &ops); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	res := make([]*entities.BankOperation, len(ops))
	for i, row := range ops {
		res[i] = row.ToDomain()
	}

	return res, nil
}
