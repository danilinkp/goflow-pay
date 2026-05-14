package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type indexer interface {
	EnsureIndexes(ctx context.Context) error
}

type Storage struct {
	User    *UserRepo
	Company *CompanyRepo
}

func Setup(ctx context.Context, db *mongo.Database) (*Storage, error) {
	s := &Storage{
		User:    NewUserRepo(db),
		Company: NewCompanyRepo(db),
	}

	repos := []indexer{
		s.User,
		s.Company,
	}

	for _, r := range repos {
		if err := r.EnsureIndexes(ctx); err != nil {
			return nil, fmt.Errorf("failed to setup indexes: %w", err)
		}
	}

	return s, nil
}
