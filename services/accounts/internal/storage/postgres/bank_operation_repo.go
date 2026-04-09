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

type BankOperationRepo struct {
	pool   *pgxpool.Pool
	getter *trmpgx.CtxGetter
}

func NewBankOperationRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *BankOperationRepo {
	return &BankOperationRepo{
		pool:   pool,
		getter: c,
	}
}

func (r *BankOperationRepo) Save(ctx context.Context, operation *entities.BankOperation) error {
	op := "BankOperationRepo.Save"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	bankOpModel := models.ToBankOperationModel(operation)
	query := `INSERT INTO bank_operations(bank_operation_id, account_id, bank_account_id, operation_type, operation_status, amount, idempotency_key, external_id, updated_at, created_at)
			 VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := conn.Exec(ctx, query,
		bankOpModel.BankOperationId,
		bankOpModel.AccountId,
		bankOpModel.BankAccountId,
		bankOpModel.OperationType,
		bankOpModel.OperationStatus,
		bankOpModel.Amount,
		bankOpModel.IdempotencyKey,
		bankOpModel.ExternalId,
		bankOpModel.UpdatedAt,
		bankOpModel.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, domain.ErrBankOperationAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *BankOperationRepo) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*entities.BankOperation, error) {
	op := "BankOperationRepo.GetByIdempotencyKey"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `SELECT bank_operation_id, account_id, bank_account_id, operation_type, operation_status, amount, idempotency_key, external_id, updated_at, created_at
 			  FROM bank_operations WHERE idempotency_key = $1`
	var bankOpModel models.BankOperationModel
	err := conn.QueryRow(ctx, query, idempotencyKey).Scan(
		&bankOpModel.BankOperationId,
		&bankOpModel.AccountId,
		&bankOpModel.BankAccountId,
		&bankOpModel.OperationType,
		&bankOpModel.OperationStatus,
		&bankOpModel.Amount,
		&bankOpModel.IdempotencyKey,
		&bankOpModel.ExternalId,
		&bankOpModel.UpdatedAt,
		&bankOpModel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrBankOperationNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return bankOpModel.ToDomain(), nil
}

func (r *BankOperationRepo) UpdateStatusAndExternalID(ctx context.Context, operationId uuid.UUID, status string, externalID string) error {
	op := "BankOperationRepo.UpdateStatusAndExternalID"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `UPDATE bank_operations SET operation_status = $1, external_id = $2 WHERE bank_operation_id = $3;`

	result, err := conn.Exec(ctx, query, status, externalID, operationId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrBankOperationNotFound
	}
	return nil
}
