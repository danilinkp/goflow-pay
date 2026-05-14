package postgres

import (
	"accounts/internal/domain"
	"accounts/internal/domain/entities"
	"accounts/internal/storage/postgres/models"
	"context"
	"errors"
	"fmt"
	postgresLib "shared/pkg/db/postgres"
	"time"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatementRepo struct {
	pool    *pgxpool.Pool
	getter  *trmpgx.CtxGetter
	factory *postgresLib.SelectFactory[models.StatementModel]
}

func NewStatementRepo(pool *pgxpool.Pool, c *trmpgx.CtxGetter) *StatementRepo {
	return &StatementRepo{
		pool:    pool,
		getter:  c,
		factory: postgresLib.NewSelectFactory[models.StatementModel](pool, c, "statements", models.StatementColumns()),
	}
}

func (r *StatementRepo) Save(ctx context.Context, statement *entities.Statement) error {
	op := "StatementRepo.Save"

	conn := r.getter.DefaultTrOrDB(ctx, r.pool)
	statementModel := models.ToStatementModel(statement)
	query := `INSERT INTO statements(statement_id, account_id, company_id, initiator_id, period_from, period_to,
			   opening_balance, closing_balance, total_debit, total_credit, currency, entries, updated_at, created_at)
			  VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	_, err := conn.Exec(ctx, query,
		statementModel.StatementId,
		statementModel.AccountId,
		statementModel.CompanyId,
		statementModel.InitiatorId,
		statementModel.PeriodFrom,
		statementModel.PeriodTo,
		statementModel.OpeningBalance,
		statementModel.ClosingBalance,
		statementModel.TotalDebit,
		statementModel.TotalCredit,
		statementModel.Currency,
		statementModel.Entries,
		statementModel.UpdatedAt,
		statementModel.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, domain.ErrStatementAlreadyExists)
		}
	}
	return nil
}

func (r *StatementRepo) GetByAccountIdAndPeriod(ctx context.Context, accountId uuid.UUID, from, to time.Time) (*entities.Statement, error) {
	op := "StatementRepo.GetByAccountIdAndPeriod"

	statementModel, err := r.factory.GetOne(ctx, "account_id = $1 AND period_from >= $2 AND period_to <= $3", accountId, from, to)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, domain.ErrStatementNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return statementModel.ToDomain(), nil
}
