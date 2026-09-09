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

type StatementRepo struct {
	collection *mongo.Collection
}

func NewStatementRepo(db *mongo.Database) *StatementRepo {
	return &StatementRepo{
		collection: db.Collection("statements"),
	}
}

func (r *StatementRepo) EnsureIndexes(ctx context.Context) error {
	op := "StatementRepo.EnsureIndexes"

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "account_id", Value: 1}, {Key: "period_from", Value: 1}, {Key: "period_to", Value: 1}},
			Options: options.Index().SetName("idx_account_id_period"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *StatementRepo) Save(ctx context.Context, statement *entities.Statement) error {
	op := "StatementRepo.Save"

	statementModel := models.ToStatementModel(statement)
	_, err := r.collection.InsertOne(ctx, statementModel)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, domain.ErrStatementAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *StatementRepo) GetByAccountIdAndPeriod(ctx context.Context, accountId uuid.UUID, from, to time.Time) (*entities.Statement, error) {
	op := "StatementRepo.GetByAccountIdAndPeriod"

	filter := bson.M{"account_id": accountId,
		"period_from": bson.M{
			"$gte": from,
		},
		"period_to": bson.M{
			"$lte": to,
		},
	}
	var statementModel models.StatementModel
	err := r.collection.FindOne(ctx, filter).Decode(&statementModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrStatementNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return statementModel.ToDomain(), nil
}
