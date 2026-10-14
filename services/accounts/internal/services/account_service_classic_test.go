package services_test

import (
	"accounts/internal/domain/entities"
	"accounts/internal/services"
	"context"
	"shared/pkg/outbox"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func TestAccountService_Classic_ReserveWithdraw_Success(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		fixedTime := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
		svc, mock := setupPgxMockAccountService(t, func() time.Time { return fixedTime })
		defer mock.Close()

		acc := NewAccountBuilder().
			WithBalance(1000).
			WithStatus(entities.ActiveStatus).
			Build()

		withdrawAmount := int64(300)
		toAccountID := uuid.New()
		txID := uuid.New()

		a.Step("Arrange: configure pgxmock expectations with exact deterministic timestamp", func(a *allure.Context) {
			mock.ExpectQuery(`SELECT (.+) FROM accounts WHERE (.+) LIMIT 1`).
				WithArgs(acc.AccountId()).
				WillReturnRows(newAccountMockRows(mock, acc))

			mock.ExpectQuery(`SELECT (.+) FROM account_operations WHERE (.+) LIMIT 1`).
				WithArgs(txID, string(entities.Withdrawal)).
				WillReturnRows(newAccountOperationMockRows(mock))

			mock.ExpectExec(`UPDATE accounts SET balance = balance \+ \$1 WHERE account_id = \$2;`).
				WithArgs(-withdrawAmount, acc.AccountId()).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))

			mock.ExpectExec(`INSERT INTO account_operations`).
				WithArgs(
					pgxmock.AnyArg(),
					acc.AccountId(),
					toAccountID,
					txID,
					string(entities.Withdrawal),
					string(entities.PendingStatus),
					withdrawAmount,
					int64(700),
					fixedTime,
					fixedTime,
				).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))
		})

		var op *entities.AccountOperation
		var err error
		a.Step("Act: execute ReserveWithdraw on AccountService with real Postgres repositories", func(a *allure.Context) {
			op, err = svc.ReserveWithdraw(ctx, acc.AccountId(), toAccountID, txID, withdrawAmount)
		})

		a.Step("Assert: verify operation created, exact timestamp preserved, and SQL expectations met", func(a *allure.Context) {
			require.NoError(t, err)
			require.NotNil(t, op)
			assert.Equal(t, withdrawAmount, op.Amount())
			assert.Equal(t, int64(700), op.BalanceAfter())
			assert.Equal(t, entities.Withdrawal, op.OperationType())
			assert.Equal(t, fixedTime, op.CreatedAt())
			assert.Equal(t, fixedTime, op.UpdatedAt())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	},
		allure.WithParentSuite("Классическая школа (Sociable)"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("ReserveWithdraw"),
		allure.WithEpic("Accounts"),
		allure.WithFeature("ReserveWithdraw"),
		allure.WithStory("Positive: Successful funds reservation for withdrawal"),
	)
}

func TestAccountService_Classic_ReserveWithdraw_ExactBalance_Success(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		svc, mock := setupPgxMockAccountService(t)
		defer mock.Close()

		initialBalance := int64(500)
		acc := NewAccountBuilder().
			WithBalance(initialBalance).
			WithStatus(entities.ActiveStatus).
			Build()

		toAccountID := uuid.New()
		txID := uuid.New()

		a.Step("Arrange: configure pgxmock for exact balance withdrawal", func(a *allure.Context) {
			mock.ExpectQuery(`SELECT (.+) FROM accounts WHERE (.+) LIMIT 1`).
				WithArgs(acc.AccountId()).
				WillReturnRows(newAccountMockRows(mock, acc))

			mock.ExpectQuery(`SELECT (.+) FROM account_operations WHERE (.+) LIMIT 1`).
				WithArgs(txID, string(entities.Withdrawal)).
				WillReturnRows(newAccountOperationMockRows(mock))

			mock.ExpectExec(`UPDATE accounts SET balance = balance \+ \$1 WHERE account_id = \$2;`).
				WithArgs(-initialBalance, acc.AccountId()).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))

			mock.ExpectExec(`INSERT INTO account_operations`).
				WithArgs(
					pgxmock.AnyArg(),
					acc.AccountId(),
					toAccountID,
					txID,
					string(entities.Withdrawal),
					string(entities.PendingStatus),
					initialBalance,
					int64(0),
					pgxmock.AnyArg(),
					pgxmock.AnyArg(),
				).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))
		})

		var op *entities.AccountOperation
		var err error
		a.Step("Act: execute ReserveWithdraw for exact balance 500", func(a *allure.Context) {
			op, err = svc.ReserveWithdraw(ctx, acc.AccountId(), toAccountID, txID, initialBalance)
		})

		a.Step("Assert: verify balance reduced to exactly 0 and SQL expectations met", func(a *allure.Context) {
			require.NoError(t, err)
			require.NotNil(t, op)
			assert.Equal(t, initialBalance, op.Amount())
			assert.Equal(t, int64(0), op.BalanceAfter())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	},
		allure.WithParentSuite("Классическая школа (Sociable)"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("ReserveWithdraw"),
		allure.WithEpic("Accounts"),
		allure.WithFeature("ReserveWithdraw"),
		allure.WithStory("Positive: Withdraw exact account balance"),
	)
}

func TestAccountService_Classic_ReserveWithdraw_Idempotency_ReturnsExistingWithoutDuplicateDeduction(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		svc, mock := setupPgxMockAccountService(t)
		defer mock.Close()

		acc := NewAccountBuilder().
			WithBalance(1000).
			WithStatus(entities.ActiveStatus).
			Build()

		txID := uuid.New()
		toAccountID := uuid.New()
		withdrawAmount := int64(300)

		a.Step("Arrange: configure pgxmock for first withdrawal call", func(a *allure.Context) {
			mock.ExpectQuery(`SELECT (.+) FROM accounts WHERE (.+) LIMIT 1`).
				WithArgs(acc.AccountId()).
				WillReturnRows(newAccountMockRows(mock, acc))

			mock.ExpectQuery(`SELECT (.+) FROM account_operations WHERE (.+) LIMIT 1`).
				WithArgs(txID, string(entities.Withdrawal)).
				WillReturnRows(newAccountOperationMockRows(mock))

			mock.ExpectExec(`UPDATE accounts SET balance = balance \+ \$1 WHERE account_id = \$2;`).
				WithArgs(-withdrawAmount, acc.AccountId()).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))

			mock.ExpectExec(`INSERT INTO account_operations`).
				WithArgs(
					pgxmock.AnyArg(),
					acc.AccountId(),
					toAccountID,
					txID,
					string(entities.Withdrawal),
					string(entities.PendingStatus),
					withdrawAmount,
					int64(700),
					pgxmock.AnyArg(),
					pgxmock.AnyArg(),
				).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))
		})

		var firstOp, secondOp *entities.AccountOperation
		var err1, err2 error
		a.Step("Act: execute ReserveWithdraw twice with identical transaction ID", func(a *allure.Context) {
			firstOp, err1 = svc.ReserveWithdraw(ctx, acc.AccountId(), toAccountID, txID, withdrawAmount)
			require.NoError(t, err1)
			require.NotNil(t, firstOp)

			mock.ExpectQuery(`SELECT (.+) FROM accounts WHERE (.+) LIMIT 1`).
				WithArgs(acc.AccountId()).
				WillReturnRows(newAccountMockRows(mock, acc))

			mock.ExpectQuery(`SELECT (.+) FROM account_operations WHERE (.+) LIMIT 1`).
				WithArgs(txID, string(entities.Withdrawal)).
				WillReturnRows(newAccountOperationMockRows(mock, firstOp))

			secondOp, err2 = svc.ReserveWithdraw(ctx, acc.AccountId(), toAccountID, txID, withdrawAmount)
		})

		a.Step("Assert: verify second call returns identical operation without duplicate deduction", func(a *allure.Context) {
			require.NoError(t, err1)
			require.NotNil(t, firstOp)
			require.NoError(t, err2)
			require.NotNil(t, secondOp)

			assert.Equal(t, firstOp.AccountOperationId(), secondOp.AccountOperationId())
			assert.Equal(t, withdrawAmount, secondOp.Amount())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	},
		allure.WithParentSuite("Классическая школа (Sociable)"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("ReserveWithdraw"),
		allure.WithEpic("Accounts"),
		allure.WithFeature("ReserveWithdraw"),
		allure.WithStory("Positive: Idempotency prevents duplicate withdrawal deduction"),
	)
}

func TestAccountService_Classic_ReserveWithdraw_InsufficientFunds_ThrowsError(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		svc, mock := setupPgxMockAccountService(t)
		defer mock.Close()

		initialBalance := int64(1000)
		acc := NewAccountBuilder().
			WithBalance(initialBalance).
			Build()

		withdrawAmount := initialBalance + 1
		toAccountID := uuid.New()
		txID := uuid.New()

		a.Step("Arrange: configure pgxmock returning balance 1000 and request 1001", func(a *allure.Context) {
			mock.ExpectQuery(`SELECT (.+) FROM accounts WHERE (.+) LIMIT 1`).
				WithArgs(acc.AccountId()).
				WillReturnRows(newAccountMockRows(mock, acc))

			mock.ExpectQuery(`SELECT (.+) FROM account_operations WHERE (.+) LIMIT 1`).
				WithArgs(txID, string(entities.Withdrawal)).
				WillReturnRows(newAccountOperationMockRows(mock))
		})

		var op *entities.AccountOperation
		var err error
		a.Step("Act: execute ReserveWithdraw exceeding account balance", func(a *allure.Context) {
			op, err = svc.ReserveWithdraw(ctx, acc.AccountId(), toAccountID, txID, withdrawAmount)
		})

		a.Step("Assert: verify ErrNotEnoughFunds returned and no balance update or operation saved", func(a *allure.Context) {
			require.Error(t, err)
			assert.Nil(t, op)
			assert.ErrorIs(t, err, services.ErrNotEnoughFunds)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	},
		allure.WithParentSuite("Классическая школа (Sociable)"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("ReserveWithdraw"),
		allure.WithEpic("Accounts"),
		allure.WithFeature("ReserveWithdraw"),
		allure.WithStory("Negative: Insufficient funds rejection"),
	)
}

func TestAccountService_Classic_SetAccountInActive_Success(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		svc, mock := setupPgxMockAccountService(t)
		defer mock.Close()

		var mother AccountMother
		acc := mother.ValidActiveAccount()

		a.Step("Arrange: configure pgxmock for account with no pending operations", func(a *allure.Context) {
			mock.ExpectQuery(`SELECT (.+) FROM accounts WHERE (.+) LIMIT 1`).
				WithArgs(acc.AccountId()).
				WillReturnRows(newAccountMockRows(mock, acc))

			mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM account_operations WHERE account_id = \$1 and operation_status = \$2\);`).
				WithArgs(acc.AccountId(), entities.PendingStatus).
				WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

			mock.ExpectExec(`UPDATE accounts SET status = \$1 WHERE account_id = \$2;`).
				WithArgs(entities.InactiveStatus, acc.AccountId()).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		})

		var updatedAcc *entities.Account
		var err error
		a.Step("Act: execute SetAccountInActive", func(a *allure.Context) {
			updatedAcc, err = svc.SetAccountInActive(ctx, acc.AccountId(), acc.CompanyId())
		})

		a.Step("Assert: verify status updated to Inactive and SQL expectations met", func(a *allure.Context) {
			require.NoError(t, err)
			require.NotNil(t, updatedAcc)
			assert.Equal(t, entities.InactiveStatus, updatedAcc.Status())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	},
		allure.WithParentSuite("Классическая школа (Sociable)"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("SetAccountInActive"),
		allure.WithEpic("Accounts"),
		allure.WithFeature("SetAccountInActive"),
		allure.WithStory("Positive: Deactivate account without pending operations"),
	)
}

func TestAccountService_Classic_SetAccountInActive_PendingOperations_ThrowsError(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		svc, mock := setupPgxMockAccountService(t)
		defer mock.Close()

		var mother AccountMother
		acc := mother.ValidActiveAccount()

		a.Step("Arrange: configure pgxmock reporting pending operations exist", func(a *allure.Context) {
			mock.ExpectQuery(`SELECT (.+) FROM accounts WHERE (.+) LIMIT 1`).
				WithArgs(acc.AccountId()).
				WillReturnRows(newAccountMockRows(mock, acc))

			mock.ExpectQuery(`SELECT EXISTS\(\s*SELECT 1 FROM account_operations WHERE account_id = \$1 and operation_status = \$2\);`).
				WithArgs(acc.AccountId(), entities.PendingStatus).
				WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		})

		var updatedAcc *entities.Account
		var err error
		a.Step("Act: execute SetAccountInActive", func(a *allure.Context) {
			updatedAcc, err = svc.SetAccountInActive(ctx, acc.AccountId(), acc.CompanyId())
		})

		a.Step("Assert: verify ErrAccountHasPendingOperations and no status update executed", func(a *allure.Context) {
			require.Error(t, err)
			assert.Nil(t, updatedAcc)
			assert.ErrorIs(t, err, services.ErrAccountHasPendingOperations)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	},
		allure.WithParentSuite("Классическая школа (Sociable)"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("SetAccountInActive"),
		allure.WithEpic("Accounts"),
		allure.WithFeature("SetAccountInActive"),
		allure.WithStory("Negative: Prevent deactivation when account has pending operations"),
	)
}

func TestAccountService_Classic_GenerateStatement_CalculatesAggregates(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		svc, mock := setupPgxMockAccountService(t)
		defer mock.Close()

		acc := NewAccountBuilder().
			WithBalance(4000).
			WithCurrency(entities.RUB).
			Build()

		baseDate := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
		periodFrom := baseDate.Add(-24 * time.Hour)
		periodTo := baseDate.Add(24 * time.Hour)

		t1 := baseDate.Add(-2 * time.Hour)
		t2 := baseDate.Add(-1 * time.Hour)

		accOp := entities.ReconstructAccountOperation(
			uuid.New(),
			acc.AccountId(),
			uuid.New(),
			uuid.New(),
			entities.Deposit,
			entities.SuccessStatus,
			2000,
			5000,
			t1,
			t1,
		)

		bankOp := entities.ReconstructBankAccountOperation(
			uuid.New(),
			acc.AccountId(),
			uuid.New(),
			uuid.New(),
			"Sberbank",
			entities.Withdrawal,
			entities.SuccessStatus,
			1000,
			4000,
			uuid.New().String(),
			"ext-1",
			t2,
			t2,
		)

		a.Step("Arrange: configure pgxmock with operations for statement period", func(a *allure.Context) {
			mock.ExpectQuery(`SELECT (.+) FROM statements WHERE (.+)`).
				WithArgs(acc.AccountId(), periodFrom, periodTo).
				WillReturnRows(newStatementMockRows(mock, nil))

			mock.ExpectQuery(`SELECT (.+) FROM accounts WHERE (.+) LIMIT 1`).
				WithArgs(acc.AccountId()).
				WillReturnRows(newAccountMockRows(mock, acc))

			mock.ExpectQuery(`SELECT (.+) FROM account_operations WHERE (.+)`).
				WithArgs(acc.AccountId(), periodFrom, periodTo).
				WillReturnRows(newAccountOperationMockRows(mock, accOp))

			mock.ExpectQuery(`SELECT (.+) FROM bank_operations WHERE (.+)`).
				WithArgs(acc.AccountId(), periodFrom, periodTo).
				WillReturnRows(newBankOperationMockRows(mock, bankOp))

			mock.ExpectExec(`INSERT INTO statements`).
				WithArgs(
					pgxmock.AnyArg(),
					acc.AccountId(),
					acc.CompanyId(),
					pgxmock.AnyArg(),
					periodFrom,
					periodTo,
					int64(3000),
					int64(4000),
					int64(2000),
					int64(1000),
					entities.RUB.String(),
					pgxmock.AnyArg(),
					pgxmock.AnyArg(),
					pgxmock.AnyArg(),
				).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))

			mock.ExpectExec(`INSERT INTO outbox`).
				WithArgs(
					pgxmock.AnyArg(),
					acc.AccountId(),
					pgxmock.AnyArg(),
					outbox.EventStatementGenerated,
					"notifications",
					pgxmock.AnyArg(),
					pgxmock.AnyArg(),
					0,
					pgxmock.AnyArg(),
					pgxmock.AnyArg(),
				).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))
		})

		var statement *entities.Statement
		var err error
		a.Step("Act: execute GenerateStatement", func(a *allure.Context) {
			statement, err = svc.GenerateStatement(ctx, acc.AccountId(), uuid.New(), acc.CompanyId(), periodFrom, periodTo)
		})

		a.Step("Assert: verify statement aggregates and SQL expectations met", func(a *allure.Context) {
			require.NoError(t, err)
			require.NotNil(t, statement)
			assert.Equal(t, int64(3000), statement.OpeningBalance())
			assert.Equal(t, int64(4000), statement.ClosingBalance())
			assert.Equal(t, int64(2000), statement.TotalDebit())
			assert.Equal(t, int64(1000), statement.TotalCredit())
			assert.Len(t, statement.Entries(), 2)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	},
		allure.WithParentSuite("Классическая школа (Sociable)"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("GenerateStatement"),
		allure.WithEpic("Accounts"),
		allure.WithFeature("Statement Generation"),
		allure.WithStory("Positive: Calculate debit, credit, opening and closing balances"),
	)
}

func TestAccountService_Classic_GenerateStatement_NoOperations_ThrowsError(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		svc, mock := setupPgxMockAccountService(t)
		defer mock.Close()

		acc := NewAccountBuilder().Build()
		periodFrom := time.Now().Add(-24 * time.Hour)
		periodTo := time.Now().Add(24 * time.Hour)

		a.Step("Arrange: configure pgxmock with zero operations in target period", func(a *allure.Context) {
			mock.ExpectQuery(`SELECT (.+) FROM statements WHERE (.+)`).
				WithArgs(acc.AccountId(), periodFrom, periodTo).
				WillReturnRows(newStatementMockRows(mock, nil))

			mock.ExpectQuery(`SELECT (.+) FROM accounts WHERE (.+) LIMIT 1`).
				WithArgs(acc.AccountId()).
				WillReturnRows(newAccountMockRows(mock, acc))

			mock.ExpectQuery(`SELECT (.+) FROM account_operations WHERE (.+)`).
				WithArgs(acc.AccountId(), periodFrom, periodTo).
				WillReturnRows(newAccountOperationMockRows(mock))

			mock.ExpectQuery(`SELECT (.+) FROM bank_operations WHERE (.+)`).
				WithArgs(acc.AccountId(), periodFrom, periodTo).
				WillReturnRows(newBankOperationMockRows(mock))
		})

		var statement *entities.Statement
		var err error
		a.Step("Act: execute GenerateStatement with no operations", func(a *allure.Context) {
			statement, err = svc.GenerateStatement(ctx, acc.AccountId(), uuid.New(), acc.CompanyId(), periodFrom, periodTo)
		})

		a.Step("Assert: verify ErrNotAnyOperations returned and SQL expectations met", func(a *allure.Context) {
			require.Error(t, err)
			assert.Nil(t, statement)
			assert.ErrorIs(t, err, services.ErrNotAnyOperations)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	},
		allure.WithParentSuite("Классическая школа (Sociable)"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("GenerateStatement"),
		allure.WithEpic("Accounts"),
		allure.WithFeature("Statement Generation"),
		allure.WithStory("Negative: Reject statement generation when no operations exist in period"),
	)
}
