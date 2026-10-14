package services_test

import (
	"accounts/internal/domain/entities"
	"accounts/internal/services"
	"accounts/internal/storage/postgres"
	"accounts/internal/storage/postgres/models"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	outboxPg "shared/pkg/outbox/repository/postgres"
	"testing"
	"time"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/require"
)

type pgxTransactor struct{}

func (pgxTransactor) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type pgxBankGateway struct{}

func (pgxBankGateway) Deposit(ctx context.Context, ba *entities.BankAccount, amount int64, key string) (string, error) {
	return "ext-" + key, nil
}
func (pgxBankGateway) Withdraw(ctx context.Context, ba *entities.BankAccount, amount int64, key string) (string, error) {
	return "ext-" + key, nil
}

func setupPgxMockAccountService(t *testing.T, now ...func() time.Time) (*services.AccountService, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	getter := trmpgx.DefaultCtxGetter
	accountRepo := postgres.NewAccountRepo(mock, getter)
	accountOpRepo := postgres.NewAccountOperationRepo(mock, getter)
	bankAccountRepo := postgres.NewBankAccountRepo(mock, getter)
	bankOpRepo := postgres.NewBankOperationRepo(mock, getter)
	statementRepo := postgres.NewStatementRepo(mock, getter)
	outboxRepo := outboxPg.NewOutboxRepo(mock, getter)

	gateway := pgxBankGateway{}
	transactor := pgxTransactor{}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	svc := services.NewAccountService(
		accountRepo,
		accountOpRepo,
		bankAccountRepo,
		bankOpRepo,
		statementRepo,
		gateway,
		outboxRepo,
		transactor,
		logger,
		now...,
	)

	return svc, mock
}

func newAccountMockRows(mock pgxmock.PgxPoolIface, acc *entities.Account) *pgxmock.Rows {
	rows := mock.NewRows(models.AccountColumns())
	if acc != nil {
		rows.AddRow(
			acc.AccountId(),
			acc.CompanyId(),
			acc.Balance(),
			acc.Currency().String(),
			acc.Status().String(),
			acc.CreatedAt(),
			acc.UpdatedAt(),
		)
	}
	return rows
}

func newAccountOperationMockRows(mock pgxmock.PgxPoolIface, ops ...*entities.AccountOperation) *pgxmock.Rows {
	rows := mock.NewRows(models.AccountOperationColumns())
	for _, op := range ops {
		rows.AddRow(
			op.AccountOperationId(),
			op.AccountId(),
			op.CounterpartyId(),
			op.TransactionId(),
			op.OperationType().String(),
			op.OperationStatus().String(),
			op.Amount(),
			op.BalanceAfter(),
			op.UpdatedAt(),
			op.CreatedAt(),
		)
	}
	return rows
}

func newBankOperationMockRows(mock pgxmock.PgxPoolIface, ops ...*entities.BankOperation) *pgxmock.Rows {
	rows := mock.NewRows(models.BankOperationColumns())
	for _, op := range ops {
		rows.AddRow(
			op.BankOperationId(),
			op.AccountId(),
			op.BankAccountId(),
			op.InitiatorId(),
			op.BankName(),
			op.OperationType().String(),
			op.OperationStatus().String(),
			op.Amount(),
			op.BalanceAfter(),
			op.IdempotencyKey(),
			op.ExternalId(),
			op.UpdatedAt(),
			op.CreatedAt(),
		)
	}
	return rows
}

func newStatementMockRows(mock pgxmock.PgxPoolIface, s *entities.Statement) *pgxmock.Rows {
	rows := mock.NewRows(models.StatementColumns())
	if s != nil {
		entriesJSON, _ := json.Marshal(s.Entries())
		rows.AddRow(
			s.StatementId(),
			s.AccountId(),
			s.CompanyId(),
			s.InitiatorId(),
			s.PeriodFrom(),
			s.PeriodTo(),
			s.OpeningBalance(),
			s.ClosingBalance(),
			s.TotalDebit(),
			s.TotalCredit(),
			s.Currency().String(),
			entriesJSON,
			s.UpdatedAt(),
			s.CreatedAt(),
		)
	}
	return rows
}
