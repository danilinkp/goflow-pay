package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Transactor struct {
	db *pgxpool.Pool
}

func NewTransactor(db *pgxpool.Pool) *Transactor {
	return &Transactor{db: db}
}

func (t *Transactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(ctx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
