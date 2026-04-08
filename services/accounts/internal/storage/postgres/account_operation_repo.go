package postgres

import (
	"accounts/internal/domain"
	"accounts/internal/domain/entities"
	"accounts/internal/storage/postgres/models"
	"context"
	"errors"
	"fmt"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountOperationRepo struct {
	pool   *pgxpool.Pool
	getter *trmpgx.CtxGetter
}

func NewAccountOperationRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *AccountOperationRepo {
	return &AccountOperationRepo{
		pool:   pool,
		getter: c,
	}
}

func (r *AccountOperationRepo) Save(ctx context.Context, accountOp *entities.AccountOperation) error {
	op := "AccountOperationRepo.Save"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	accountOpModel := models.ToAccountOperationModel(accountOp)
	query := `INSERT INTO account_operations(operation_id, account_id, transaction_id, operation_type, operation_status, amount, updated_at, created_at)
			 VALUES($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := conn.Exec(ctx, query,
		accountOpModel.OperationId,
		accountOpModel.AccountId,
		accountOpModel.TransactionId,
		accountOpModel.OperationType,
		accountOpModel.OperationStatus,
		accountOpModel.Amount,
		accountOpModel.UpdatedAt,
		accountOpModel.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, domain.ErrAccountOperationAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *AccountOperationRepo) GetByTransactionIdAndType(ctx context.Context, transactionId uuid.UUID, operationType string) (*entities.AccountOperation, error) {
	op := "AccountOperationRepo.GetByTransactionIdAndType"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `SELECT operation_id, account_id, transaction_id, operation_type, operation_status, amount, updated_at, created_at
 			  FROM account_operations WHERE transaction_id = $1 AND operation_type = $2`
	var accountOpModel models.AccountOperationModel
	err := conn.QueryRow(ctx, query, transactionId, operationType).Scan(
		&accountOpModel.OperationId,
		&accountOpModel.AccountId,
		&accountOpModel.TransactionId,
		&accountOpModel.OperationType,
		&accountOpModel.OperationStatus,
		&accountOpModel.Amount,
		&accountOpModel.UpdatedAt,
		&accountOpModel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrAccountOperationNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return accountOpModel.ToDomain(), nil
}

func (r *AccountOperationRepo) GetByTransactionId(ctx context.Context, transactionId uuid.UUID) ([]*entities.AccountOperation, error) {
	op := "AccountOperationRepo.GetByTransactionId"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `SELECT operation_id, account_id, transaction_id, operation_type, operation_status, amount, updated_at, created_at
			  FROM account_operations WHERE transaction_id = $1`

	rows, err := conn.Query(ctx, query, transactionId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	ops, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.AccountOperationModel])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	res := make([]*entities.AccountOperation, len(ops))
	for i, row := range ops {
		res[i] = row.ToDomain()
	}

	return res, nil
}

func (r *AccountOperationRepo) UpdateStatus(ctx context.Context, operation uuid.UUID, status string) error {
	op := "AccountOperationRepo.UpdateStatus"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `UPDATE account_operations SET operation_status = $1 WHERE account_operation_id = $2;`

	result, err := conn.Exec(ctx, query, status, operation)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrAccountOperationNotFound
	}
	return nil
}

func (r *AccountOperationRepo) ConfirmAllByTransactionId(ctx context.Context, transactionId uuid.UUID) error {
	op := "AccountOperationRepo.ConfirmAllByTransactionId"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `UPDATE account_operations SET operation_status = $1 WHERE transaction_id = $2;`

	result, err := conn.Exec(ctx, query, entities.SuccessStatus, transactionId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrAccountOperationNotFound
	}
	return nil
}

func (r *AccountOperationRepo) HasPendingByAccountId(ctx context.Context, accountId uuid.UUID) (bool, error) {
	op := "AccountOperationRepo.HasPendingByAccountId"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	st := entities.PendingStatus
	query := `SELECT EXISTS(
			  SELECT 1 FROM account_operations WHERE account_id = $1 and operation_type = $2);
`
	var exists bool
	err := conn.QueryRow(ctx, query, accountId, st).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}
