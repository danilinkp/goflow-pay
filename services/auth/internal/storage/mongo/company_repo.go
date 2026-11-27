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

type CompanyRepo struct {
	collection *mongo.Collection
}

func NewCompanyRepo(db *mongo.Database) *CompanyRepo {
	return &CompanyRepo{
		collection: db.Collection("companies"),
	}
}

func (r *CompanyRepo) EnsureIndexes(ctx context.Context) error {
	op := "CompanyRepo.EnsureIndexes"

	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "invite_code", Value: 1}},
		Options: options.Index().
			SetUnique(true).
			SetName("idx_invite_code"),
	}

	_, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *CompanyRepo) Save(ctx context.Context, company *entities.Company) error {
	op := "CompanyRepo.Save"

	companyModel := models.ToCompanyModel(company)

	_, err := r.collection.InsertOne(ctx, companyModel)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%s: %w", op, domain.ErrCompanyAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *CompanyRepo) GetById(ctx context.Context, companyId uuid.UUID) (*entities.Company, error) {
	op := "CompanyRepo.GetById"

	filter := bson.M{"_id": companyId}

	var companyModel models.CompanyModel
	err := r.collection.FindOne(ctx, filter).Decode(&companyModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrCompanyNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return companyModel.ToDomain(), nil
}

func (r *CompanyRepo) GetByInviteCode(ctx context.Context, inviteCode string) (*entities.Company, error) {
	op := "CompanyRepo.GetByInviteCode"

	filter := bson.M{"invite_code": inviteCode}
	var companyModel models.CompanyModel
	err := r.collection.FindOne(ctx, filter).Decode(&companyModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrCompanyNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return companyModel.ToDomain(), nil
}

func (r *CompanyRepo) GetInviteCodeById(ctx context.Context, companyId uuid.UUID) (string, error) {
	op := "CompanyRepo.GetInviteCodeById"

	filter := bson.M{"_id": companyId}
	projection := bson.M{"invite_code": 1, "_id": 0}

	var result struct {
		InviteCode string `bson:"invite_code"`
	}

	err := r.collection.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", fmt.Errorf("%s: %w", op, domain.ErrCompanyNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return result.InviteCode, nil
}

func (r *CompanyRepo) GetAll(ctx context.Context, search string, limit, offset int) ([]*entities.Company, int, error) {
	op := "CompanyRepo.GetAll"
	filter := bson.M{}
	if search != "" {
		filter["name"] = bson.M{"$regex": search, "$options": "i"}
	}
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}
	if offset > 0 {
		findOpts.SetSkip(int64(offset))
	}
	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	defer cursor.Close(ctx)
	var modelsList []models.CompanyModel
	if err := cursor.All(ctx, &modelsList); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	res := make([]*entities.Company, len(modelsList))
	for i, m := range modelsList {
		res[i] = m.ToDomain()
	}
	return res, int(total), nil
}
