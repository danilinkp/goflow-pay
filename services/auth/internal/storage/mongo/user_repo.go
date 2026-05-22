package mongo

import (
	"auth/internal/domain"
	"auth/internal/domain/entities"
	"auth/internal/storage/mongo/models"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type UserRepo struct {
	collection *mongo.Collection
}

func NewUserRepo(db *mongo.Database) *UserRepo {
	return &UserRepo{
		collection: db.Collection("users"),
	}
}

func (r *UserRepo) EnsureIndexes(ctx context.Context) error {
	op := "UserRepo.EnsureIndexes"

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_user_email"),
		},
		{
			Keys:    bson.D{{Key: "company_id", Value: 1}},
			Options: options.Index().SetName("idx_user_company_id"),
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("%s: ensure indexes: %v", op, err)
	}
	return nil
}

func (r *UserRepo) Save(ctx context.Context, user *entities.User) error {
	op := "UserRepo.Save"

	userModel := models.ToUserModel(user)
	_, err := r.collection.InsertOne(ctx, userModel)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, domain.ErrUserAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *UserRepo) GetById(ctx context.Context, userId uuid.UUID) (*entities.User, error) {
	op := "UserRepo.GetById"

	filter := bson.M{"_id": userId}
	var userModel models.UserModel
	err := r.collection.FindOne(ctx, filter).Decode(&userModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return userModel.ToDomain(), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	op := "UserRepo.GetByEmail"

	filter := bson.M{"email": email}

	var userModel models.UserModel
	err := r.collection.FindOne(ctx, filter).Decode(&userModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return userModel.ToDomain(), nil
}

func (r *UserRepo) GetByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.User, error) {
	op := "UserRepo.GetByCompanyId"
	filter := bson.M{"company_id": companyId}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var userModels []models.UserModel
	if err = cursor.All(ctx, &userModels); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	res := make([]*entities.User, len(userModels))
	for i, user := range userModels {
		res[i] = user.ToDomain()
	}

	return res, nil
}
