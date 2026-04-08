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

type AccountRepo struct {
	pool   *pgxpool.Pool
	getter *trmpgx.CtxGetter
}

func NewAccountRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *AccountRepo {
	return &AccountRepo{
		pool:   pool,
		getter: c,
	}
}

func (r *AccountRepo) Save(ctx context.Context, account *entities.Account) error {
	op := "AccountRepo.Save"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	accountModel := models.ToAccountModel(account)

	query := `INSERT INTO account(account_id, company_id, balance, currency, status, updated_at, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7);`

	_, err := conn.Exec(ctx, query,
		accountModel.AccountId,
		accountModel.CompanyId,
		accountModel.Balance,
		accountModel.Currency,
		accountModel.Status,
		accountModel.UpdatedAt,
		accountModel.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, domain.ErrAccountAlreadyExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *AccountRepo) GetById(ctx context.Context, id uuid.UUID) (*entities.Account, error) {
	op := "AccountRepo.GetById"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `SELECT account_id, company_id, balance, currency, status, created_at, updated_at
			  FROM account WHERE account_id = $1;`

	var accountModel models.AccountModel
	err := conn.QueryRow(ctx, query, id).Scan(
		&accountModel.AccountId,
		&accountModel.CompanyId,
		&accountModel.Balance,
		&accountModel.Currency,
		&accountModel.Status,
		&accountModel.CreatedAt,
		&accountModel.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return accountModel.ToDomain(), nil
}

func (r *AccountRepo) GetByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.Account, error) {
	op := "AccountRepo.GetByCompanyId"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `SELECT account_id, company_id, balance, currency, status, created_at, updated_at
			  FROM account WHERE company_id = $1;`

	rows, err := conn.Query(ctx, query, companyId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	accounts, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.AccountModel])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	res := make([]*entities.Account, len(accounts))
	for i, row := range accounts {
		res[i] = row.ToDomain()
	}

	return res, nil
}

func (r *AccountRepo) UpdateBalance(ctx context.Context, accountId uuid.UUID, amount int64) error {
	op := "AccountRepo.UpdateBalance"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `UPDATE accounts SET balance = balanc + $1 WHERE account_id = $2;`

	result, err := conn.Exec(ctx, query, amount, accountId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrAccountNotFound
	}
	return nil
}

func (r *AccountRepo) UpdateStatus(ctx context.Context, accountId uuid.UUID, status entities.AccountStatus) error {
	op := "AccountRepo.UpdateStatus"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	query := `UPDATE accounts SET status = $1 WHERE account_id = $2;`

	result, err := conn.Exec(ctx, query, status, accountId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrAccountNotFound
	}
	return nil
}
