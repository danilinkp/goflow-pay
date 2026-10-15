//go:build integration

package services_test

import (
	"accounts/internal/domain/entities"
	"accounts/internal/services"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func TestAccountService_RollbackAndCompensation_Integration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		fixedTime := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
		svc, pool := setupIntegrationAccountService(t, func() time.Time { return fixedTime })

		senderAcc := NewAccountBuilder().
			WithBalance(5000).
			WithStatus(entities.ActiveStatus).
			WithCurrency(entities.RUB).
			Build()

		receiverAcc := NewAccountBuilder().
			WithBalance(1000).
			WithStatus(entities.ActiveStatus).
			WithCurrency(entities.RUB).
			Build()

		withdrawAmount := int64(2000)
		txID := uuid.New()

		// Arrange (п. 17b методички: один раз в начале теста)
		a.Step("Arrange: заведение счетов в PostgreSQL с начальным балансом 5 000 руб.", func(a *allure.Context) {
			insertTestAccount(t, ctx, pool, senderAcc)
			insertTestAccount(t, ctx, pool, receiverAcc)

			var initialBalance int64
			err := pool.QueryRow(ctx, "SELECT balance FROM accounts WHERE account_id = $1", senderAcc.AccountId()).
				Scan(&initialBalance)
			require.NoError(t, err)
			assert.Equal(t, int64(5000), initialBalance)
		})

		// Act 1: Первичное резервирование списания
		a.Step("Act 1: первичное резервирование 2 000 руб. через ReserveWithdraw", func(a *allure.Context) {
			_, err := svc.ReserveWithdraw(ctx, senderAcc.AccountId(), receiverAcc.AccountId(), txID, withdrawAmount)
			require.NoError(t, err)
		})

		// Assert 1: Проверка списания в базе
		a.Step("Assert 1: проверка уменьшения баланса до 3 000 руб. и создания pending-операции", func(a *allure.Context) {
			var balance int64
			err := pool.QueryRow(ctx, "SELECT balance FROM accounts WHERE account_id = $1", senderAcc.AccountId()).
				Scan(&balance)
			require.NoError(t, err)
			assert.Equal(t, int64(3000), balance)

			var opStatus string
			err = pool.QueryRow(ctx, "SELECT operation_status FROM account_operations WHERE transaction_id = $1", txID).
				Scan(&opStatus)
			require.NoError(t, err)
			assert.Equal(t, string(entities.PendingStatus), opStatus)
		})

		// Act 2: Компенсирующий откат транзакции (CancelOperation)
		a.Step("Act 2: выполнение компенсирующего отката через CancelOperation", func(a *allure.Context) {
			err := svc.CancelOperation(ctx, txID)
			require.NoError(t, err)
		})

		// Assert 2: Проверка возврата средств и перехода статуса в failed в PostgreSQL
		a.Step("Assert 2: проверка восстановления баланса до 5 000 руб. и статуса failed в БД", func(a *allure.Context) {
			var restoredBalance int64
			err := pool.QueryRow(ctx, "SELECT balance FROM accounts WHERE account_id = $1", senderAcc.AccountId()).
				Scan(&restoredBalance)
			require.NoError(t, err)
			assert.Equal(t, int64(5000), restoredBalance, "баланс должен полностью восстановиться в БД")

			var failedStatus string
			err = pool.QueryRow(ctx, "SELECT operation_status FROM account_operations WHERE transaction_id = $1", txID).
				Scan(&failedStatus)
			require.NoError(t, err)
			assert.Equal(t, string(entities.FailedStatus), failedStatus, "статус операции должен стать failed")
		})

		// Act 3: Попытка списания суммы, превышающей баланс
		var overdrawErr error
		invalidTxID := uuid.New()
		a.Step("Act 3: попытка списать 10 000 руб. при балансе 5 000 руб.", func(a *allure.Context) {
			_, overdrawErr = svc.ReserveWithdraw(ctx, senderAcc.AccountId(), receiverAcc.AccountId(), invalidTxID, 10000)
		})

		// Assert 3: Проверка ошибки и отсутствия побочных эффектов в базе данных
		a.Step("Assert 3: проверка ошибки ErrNotEnoughFunds и неизменности баланса в БД", func(a *allure.Context) {
			require.Error(t, overdrawErr)
			assert.ErrorIs(t, overdrawErr, services.ErrNotEnoughFunds)

			var untouchedBalance int64
			err := pool.QueryRow(ctx, "SELECT balance FROM accounts WHERE account_id = $1", senderAcc.AccountId()).
				Scan(&untouchedBalance)
			require.NoError(t, err)
			assert.Equal(t, int64(5000), untouchedBalance, "баланс не должен измениться при ошибке валидации")

			var count int
			err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM account_operations WHERE transaction_id = $1", invalidTxID).
				Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 0, count, "в БД не должно появиться записей для отклонённой операции")
		})
	},
		allure.WithParentSuite("Интеграционные тесты"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("Rollback & Compensation"),
		allure.WithEpic("Управление счетами"),
		allure.WithFeature("Компенсация и транзакционный откат"),
		allure.WithStory("Откат списания с восстановлением баланса и защита от перерасхода"),
	)
}
