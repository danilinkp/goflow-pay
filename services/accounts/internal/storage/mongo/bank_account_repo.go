package mongo

import (
	"accounts/internal/domain"
	"accounts/internal/domain/entities"
	"accounts/internal/storage/mongo/models"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type BankAccountRepo struct {
	collection *mongo.Collection
}

func NewBankAccountRepo(db *mongo.Database) *BankAccountRepo {
	return &BankAccountRepo{
		collection: db.Collection("bank_accounts"),
	}
}

func (r *BankAccountRepo) EnsureIndexes(ctx context.Context) error {
	op := "BankAccountRepo.EnsureIndexes"

	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "company_id", Value: 1}},
		Options: options.Index().SetName("idx_company_id"),
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *BankAccountRepo) Save(ctx context.Context, bankAccount *entities.BankAccount) error {
	op := "BankAccountRepo.Save"

	bankAccountModel := models.ToBankAccountModel(bankAccount)
	_, err := r.collection.InsertOne(ctx, bankAccountModel)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, domain.ErrBankAccountAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *BankAccountRepo) GetById(ctx context.Context, id uuid.UUID) (*entities.BankAccount, error) {
	op := "BankAccountRepo.GetById"

	filter := bson.M{"_id": id}
	var bankAccountModel models.BankAccountModel
	err := r.collection.FindOne(ctx, filter).Decode(&bankAccountModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrBankAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return bankAccountModel.ToDomain(), nil
}

func (r *BankAccountRepo) GetByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.BankAccount, error) {
	op := "AccountRepo.GetByCompanyId"

	filter := bson.M{"company_id": companyId}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var bankAccounts []models.BankAccountModel
	if err = cursor.All(ctx, &bankAccounts); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	res := make([]*entities.BankAccount, len(bankAccounts))
	for i, row := range bankAccounts {
		res[i] = row.ToDomain()
	}

	return res, nil
}
