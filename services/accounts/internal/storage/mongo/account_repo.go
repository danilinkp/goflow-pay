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

type AccountRepo struct {
	collection *mongo.Collection
}

func NewAccountRepo(db *mongo.Database) *AccountRepo {
	return &AccountRepo{
		collection: db.Collection("accounts"),
	}
}

func (r *AccountRepo) EnsureIndexes(ctx context.Context) error {
	op := "AccountRepo.EnsureIndexes"

	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{"company_id", 1}},
		Options: options.Index().SetName("idx_company_id"),
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *AccountRepo) Save(ctx context.Context, account *entities.Account) error {
	op := "AccountRepo.Save"

	accountModel := models.ToAccountModel(account)

	_, err := r.collection.InsertOne(ctx, accountModel)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, domain.ErrAccountAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *AccountRepo) GetById(ctx context.Context, id uuid.UUID) (*entities.Account, error) {
	op := "AccountRepo.GetById"

	filter := bson.M{"_id": id}
	var accountModel models.AccountModel
	err := r.collection.FindOne(ctx, filter).Decode(&accountModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return accountModel.ToDomain(), nil
}

func (r *AccountRepo) GetByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.Account, error) {
	op := "AccountRepo.GetByCompanyId"

	filter := bson.M{"company_id": companyId}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var accounts []*models.AccountModel
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	res := make([]*entities.Account, len(accounts))
	for i, row := range accounts {
		res[i] = row.ToDomain()
	}

	return res, nil
}

func (r *AccountRepo) UpdateBalance(ctx context.Context, accountId uuid.UUID, amount int64) error {
	op := "AccountRepo.UpdateBalance"

	filter := bson.M{"_id": accountId}
	update := bson.M{
		"$inc": bson.M{"balance": amount},
		"$set": bson.M{"updated_at": time.Now().UTC()},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%s: %w", op, domain.ErrAccountNotFound)
	}
	return nil
}

func (r *AccountRepo) UpdateStatus(ctx context.Context, accountId uuid.UUID, status entities.AccountStatus) error {
	op := "AccountRepo.UpdateStatus"

	filter := bson.M{"_id": accountId}
	update := bson.M{"$set": bson.M{"status": status, "updated_at": time.Now().UTC()}}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("%s: %w", op, domain.ErrAccountNotFound)
	}
	return nil
}
