package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type RepositoryFactory struct {
	userRepo    *UserRepo
	companyRepo *CompanyRepo
}

func NewRepositoryFactory(ctx context.Context, db *mongo.Database) (*RepositoryFactory, error) {
	userRepo := NewUserRepo(db)
	companyRepo := NewCompanyRepo(db)

	if err := userRepo.EnsureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("ensure user indexes: %w", err)
	}
	if err := companyRepo.EnsureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("ensure company indexes: %w", err)
	}

	return &RepositoryFactory{
		userRepo:    userRepo,
		companyRepo: companyRepo,
	}, nil
}

func (f *RepositoryFactory) UserRepo() *UserRepo       { return f.userRepo }
func (f *RepositoryFactory) CompanyRepo() *CompanyRepo { return f.companyRepo }
