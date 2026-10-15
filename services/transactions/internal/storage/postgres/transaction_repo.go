package postgres

import (
	"context"
	"errors"
	"fmt"
	postgresLib "shared/pkg/db/postgres"
	"strings"
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
	pool    *pgxpool.Pool
	getter  *trmpgx.CtxGetter
	factory *postgresLib.SelectFactory[models.TransactionModel]
}

func (r *TransactionRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

func NewTransactionRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *TransactionRepo {
	return &TransactionRepo{
		pool:    pool,
		getter:  c,
		factory: postgresLib.NewSelectFactory[models.TransactionModel](pool, c, "transactions", models.TransactionColumns()),
	}
}

func (r *TransactionRepo) Save(ctx context.Context, transaction *entities.Transaction) error {
	op := "TransactionService.Save"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `INSERT INTO transactions (transaction_id, initiator_id, from_account_id, to_account_id, amount, currency, idempotency_key, status, updated_at, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	txModel := models.ToTransactionModel(transaction)
	_, err := conn.Exec(ctx, query,
		txModel.TransactionId,
		txModel.InitiatorId,
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

	txModel, err := r.factory.GetOne(ctx, "transaction_id = $1", id)
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

	txModel, err := r.factory.GetOne(ctx, "idempotency_key = $1", idempotencyKey)
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

	transactions, err := r.factory.List(ctx, "from_account_id = $1 or to_account_id = $1 ORDER BY created_at DESC", accountId)
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

	threshold := time.Now().Add(-olderThan)
	transactions, err := r.factory.List(ctx, "status = ANY($1) AND updated_at < $2", statuses, threshold)
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

func (r *TransactionRepo) List(ctx context.Context, accountIds []uuid.UUID, status string, currency string, limit, offset int) ([]*entities.Transaction, int, error) {
	op := "TransactionRepo.List"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := "SELECT transaction_id, from_account_id, to_account_id, initiator_id, amount, currency, status, idempotency_key, created_at, updated_at FROM transactions"
	countQuery := "SELECT COUNT(*) FROM transactions"

	var conditions []string
	var args []any

	if len(accountIds) > 0 {
		args = append(args, accountIds)
		conditions = append(conditions, fmt.Sprintf("(from_account_id = ANY($%d) OR to_account_id = ANY($%d))", len(args), len(args)))
	}
	if status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if currency != "" {
		args = append(args, currency)
		conditions = append(conditions, fmt.Sprintf("currency = $%d", len(args)))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery += whereClause
	var total int
	if err := conn.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}

	query += whereClause + " ORDER BY created_at DESC"
	if limit > 0 {
		args = append(args, limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var res []*entities.Transaction
	for rows.Next() {
		var m models.TransactionModel
		if err := rows.Scan(
			&m.TransactionId,
			&m.FromAccountId,
			&m.ToAccountId,
			&m.InitiatorId,
			&m.Amount,
			&m.Currency,
			&m.Status,
			&m.IdempotencyKey,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("%s: %w", op, err)
		}
		res = append(res, m.ToDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	return res, total, nil
}
