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

type BankAccountRepo struct {
	pool   *pgxpool.Pool
	getter *trmpgx.CtxGetter
}

func NewBankAccountRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *BankAccountRepo {
	return &BankAccountRepo{
		pool:   pool,
		getter: c,
	}
}

func (r *BankAccountRepo) Save(ctx context.Context, bankAccount *entities.BankAccount) error {
	op := "BankAccountRepo.Save"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	bankAccountModel := models.ToBankAccountModel(bankAccount)
	query := `INSERT INTO bank_accounts(bank_account_id, company_id, name, bic, settlement_account, currency, updated_at, created_at)
		      VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := conn.Exec(ctx, query,
		bankAccountModel.BankAccountId,
		bankAccountModel.CompanyId,
		bankAccountModel.Name,
		bankAccountModel.Bic,
		bankAccountModel.SettlementAccount,
		bankAccountModel.Currency,
		bankAccountModel.UpdatedAt,
		bankAccountModel.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, domain.ErrBankAccountAlreadyExists)
		}
	}

	return nil
}

func (r *BankAccountRepo) GetById(ctx context.Context, id uuid.UUID) (*entities.BankAccount, error) {
	op := "BankAccountRepo.GetById"
	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	var bankAccountModel models.BankAccountModel
	query := `SELECT bank_account_id, company_id, name, bic, settlement_account, currency, updated_at, created_at
			FROM bank_accounts WHERE bank_account_id = $1`
	err := conn.QueryRow(ctx, query, id).Scan(
		&bankAccountModel.BankAccountId,
		&bankAccountModel.CompanyId,
		&bankAccountModel.Name,
		&bankAccountModel.Bic,
		&bankAccountModel.SettlementAccount,
		&bankAccountModel.Currency,
		&bankAccountModel.UpdatedAt,
		&bankAccountModel.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrBankAccountNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return bankAccountModel.ToDomain(), nil
}

func (r *BankAccountRepo) GetByCompanyId(ctx context.Context, companyId uuid.UUID) ([]*entities.BankAccount, error) {
	op := "AccountRepo.GetByCompanyId"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)

	query := `SELECT bank_account_id, company_id, name, bic, settlement_account, currency, updated_at, created_at
			  FROM account WHERE company_id = $1;`

	rows, err := conn.Query(ctx, query, companyId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	bankAccounts, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.BankAccountModel])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	res := make([]*entities.BankAccount, len(bankAccounts))
	for i, row := range bankAccounts {
		res[i] = row.ToDomain()
	}

	return res, nil
}
