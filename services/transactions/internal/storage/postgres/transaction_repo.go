package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
	"transactions/internal/domain"
	"transactions/internal/domain/entities"
	"transactions/internal/storage/postgres/models"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepo struct {
	pool   *pgxpool.Pool
	getter *trmpgx.CtxGetter
}

func NewTransactionRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *TransactionRepo {
	return &TransactionRepo{
		pool:   pool,
		getter: c,
	}
}

func (r *TransactionRepo) Save(ctx context.Context, transaction *entities.Transaction) error {
	op := "TransactionService.Save"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `INSERT INTO transactions (transaction_id, from_account_id, to_account_id, amount, currency, idempotency_key, status, updated_at, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	txModel := models.ToTransactionModel(transaction)
	_, err := conn.Exec(ctx, query,
		txModel.TransactionId,
		txModel.FromAccountId,
		txModel.ToAccountId,
		txModel.Amount,
		txModel.Currency,
		txModel.IdempotencyKey,
		txModel.Status,
		txModel.UpdatedAt,
		txModel.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, domain.ErrTransactionAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *TransactionRepo) GetById(ctx context.Context, id uuid.UUID) (*entities.Transaction, error) {
	op := "TransactionService.GetById"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `SELECT transaction_id, from_account_id, to_account_id, amount, currency, idempotency_key, status, updated_at, created_at
			  FROM transactions WHERE transaction_id = $1`

	var txModel models.TransactionModel
	err := conn.QueryRow(ctx, query, id).Scan(
		&txModel.TransactionId,
		&txModel.FromAccountId,
		&txModel.ToAccountId,
		&txModel.Amount,
		&txModel.Currency,
		&txModel.IdempotencyKey,
		&txModel.Status,
		&txModel.UpdatedAt,
		&txModel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrTransactionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return txModel.ToDomain(), nil
}

func (r *TransactionRepo) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*entities.Transaction, error) {
	op := "TransactionService.GetByIdempotencyKey"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `SELECT transaction_id, from_account_id, to_account_id, amount, currency, idempotency_key, status, updated_at, created_at
			  FROM transactions WHERE idempotency_key = $1`

	var txModel models.TransactionModel
	err := conn.QueryRow(ctx, query, idempotencyKey).Scan(
		&txModel.TransactionId,
		&txModel.FromAccountId,
		&txModel.ToAccountId,
		&txModel.Amount,
		&txModel.Currency,
		&txModel.IdempotencyKey,
		&txModel.Status,
		&txModel.UpdatedAt,
		&txModel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrTransactionNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return txModel.ToDomain(), nil
}

func (r *TransactionRepo) GetByAccountId(ctx context.Context, accountId uuid.UUID) ([]*entities.Transaction, error) {
	op := "TransactionService.GetByAccountId"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `SELECT transaction_id, from_account_id, to_account_id, amount, currency, idempotency_key, status, updated_at, created_at
			  FROM transactions WHERE from_account_id = $1 or to_account_id = $1
			  ORDER BY created_at DESC;`

	rows, err := conn.Query(ctx, query, accountId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	transactions, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.TransactionModel])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	res := make([]*entities.Transaction, len(transactions))
	for i, tx := range transactions {
		res[i] = tx.ToDomain()
	}
	return res, nil
}

func (r *TransactionRepo) GetStale(ctx context.Context, olderThan time.Duration, statuses []string) ([]*entities.Transaction, error) {
	op := "TransactionService.GetStale"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `SELECT transaction_id, from_account_id, to_account_id, amount, currency, idempotency_key, status, updated_at, created_at
			  FROM transactions WHERE WHERE status = ANY($1) AND updated_at < $2;`

	rows, err := conn.Query(ctx, query, statuses)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	transactions, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.TransactionModel])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	res := make([]*entities.Transaction, len(transactions))
	for i, tx := range transactions {
		res[i] = tx.ToDomain()
	}
	return res, nil
}

func (r *TransactionRepo) UpdateStatus(ctx context.Context, txId uuid.UUID, status string) error {
	op := "TransactionService.UpdateStatus"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `UPDATE transactions SET status = $1 WHERE transaction_id = $2`

	result, err := conn.Exec(ctx, query, status, txId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrTransactionNotFound
	}
	return nil
}
