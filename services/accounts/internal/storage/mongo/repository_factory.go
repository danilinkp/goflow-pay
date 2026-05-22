package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Repositories struct {
	Account          *AccountRepo
	AccountOperation *AccountOperationRepo
	BankAccount      *BankAccountRepo
	BankOperation    *BankOperationRepo
	Statement        *StatementRepo
}

func NewRepositories(db *mongo.Database) *Repositories {
	return &Repositories{
		Account:          NewAccountRepo(db),
		AccountOperation: NewAccountOperationRepo(db),
		BankAccount:      NewBankAccountRepo(db),
		BankOperation:    NewBankOperationRepo(db),
		Statement:        NewStatementRepo(db),
	}
}

func (r *Repositories) EnsureIndexes(ctx context.Context) error {
	repos := []interface {
		EnsureIndexes(ctx context.Context) error
	}{
		r.Account,
		r.AccountOperation,
		r.BankAccount,
		r.BankOperation,
		r.Statement,
	}

	for _, repo := range repos {
		if err := repo.EnsureIndexes(ctx); err != nil {
			return fmt.Errorf("EnsureIndexes: %w", err)
		}
	}

	return nil
}
