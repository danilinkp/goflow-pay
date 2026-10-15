//go:build integration

package services_test

import (
	"accounts/internal/domain/entities"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func TestAccountService_ReserveWithdraw_Integration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		fixedTime := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
		svc, pool := setupIntegrationAccountService(t, func() time.Time { return fixedTime })

		senderAcc := NewAccountBuilder().
			WithBalance(10000).
			WithStatus(entities.ActiveStatus).
			WithCurrency(entities.RUB).
			Build()

		receiverAcc := NewAccountBuilder().
			WithBalance(2000).
			WithStatus(entities.ActiveStatus).
			WithCurrency(entities.RUB).
			Build()

		withdrawAmount := int64(2500)
		txID := uuid.New()

		// Arrange (один раз в начале теста согласно п. 17b методички)
		a.Step("Arrange: подготовка начального состояния счетов (отправитель и контрагент) в реальном PostgreSQL", func(a *allure.Context) {
			insertTestAccount(t, ctx, pool, senderAcc)
			insertTestAccount(t, ctx, pool, receiverAcc)

			var dbBalance int64
			var dbStatus string
			err := pool.QueryRow(ctx, "SELECT balance, status FROM accounts WHERE account_id = $1", senderAcc.AccountId()).
				Scan(&dbBalance, &dbStatus)
			require.NoError(t, err)
			assert.Equal(t, int64(10000), dbBalance)
			assert.Equal(t, string(entities.ActiveStatus), dbStatus)
		})

		// Act 1: Резервирование списания
		var accOp *entities.AccountOperation
		var reserveErr error
		a.Step("Act 1: вызов ReserveWithdraw через AccountService", func(a *allure.Context) {
			accOp, reserveErr = svc.ReserveWithdraw(ctx, senderAcc.AccountId(), receiverAcc.AccountId(), txID, withdrawAmount)
		})

		// Assert 1: Проверка результатов резервирования в домене и напрямую в PostgreSQL
		a.Step("Assert 1: проверка изменения баланса и создания операции в PostgreSQL", func(a *allure.Context) {
			require.NoError(t, reserveErr)
			require.NotNil(t, accOp)

			assert.Equal(t, senderAcc.AccountId(), accOp.AccountId())
			assert.Equal(t, receiverAcc.AccountId(), accOp.CounterpartyId())
			assert.Equal(t, txID, accOp.TransactionId())
			assert.Equal(t, entities.Withdrawal, accOp.OperationType())
			assert.Equal(t, entities.PendingStatus, accOp.OperationStatus())
			assert.Equal(t, withdrawAmount, accOp.Amount())
			assert.Equal(t, int64(7500), accOp.BalanceAfter())

			// Прямая верификация физического состояния в БД
			var currentBalance int64
			err := pool.QueryRow(ctx, "SELECT balance FROM accounts WHERE account_id = $1", senderAcc.AccountId()).
				Scan(&currentBalance)
			require.NoError(t, err)
			assert.Equal(t, int64(7500), currentBalance, "баланс в базе данных должен уменьшиться на сумму резервирования")

			var dbOpType, dbOpStatus string
			var dbAmount, dbBalanceAfter int64
			err = pool.QueryRow(ctx, `
				SELECT operation_type, operation_status, amount, balance_after 
				FROM account_operations 
				WHERE transaction_id = $1 AND operation_type = $2`,
				txID, string(entities.Withdrawal),
			).Scan(&dbOpType, &dbOpStatus, &dbAmount, &dbBalanceAfter)
			require.NoError(t, err)
			assert.Equal(t, string(entities.Withdrawal), dbOpType)
			assert.Equal(t, string(entities.PendingStatus), dbOpStatus)
			assert.Equal(t, withdrawAmount, dbAmount)
			assert.Equal(t, int64(7500), dbBalanceAfter)
		})

		// Act 2: Подтверждение операции
		var confirmErr error
		a.Step("Act 2: подтверждение операции через ConfirmOperation", func(a *allure.Context) {
			confirmErr = svc.ConfirmOperation(ctx, txID)
		})

		// Assert 2: Проверка финального статуса операции в PostgreSQL
		a.Step("Assert 2: проверка смены статуса операции на success в PostgreSQL", func(a *allure.Context) {
			require.NoError(t, confirmErr)

			var finalOpStatus string
			err := pool.QueryRow(ctx, `
				SELECT operation_status 
				FROM account_operations 
				WHERE transaction_id = $1`,
				txID,
			).Scan(&finalOpStatus)
			require.NoError(t, err)
			assert.Equal(t, string(entities.SuccessStatus), finalOpStatus, "статус операции в БД должен обновиться до success")

			// Финальный баланс остается зафиксированным
			var finalBalance int64
			err = pool.QueryRow(ctx, "SELECT balance FROM accounts WHERE account_id = $1", senderAcc.AccountId()).
				Scan(&finalBalance)
			require.NoError(t, err)
			assert.Equal(t, int64(7500), finalBalance)
		})
	},
		allure.WithParentSuite("Интеграционные тесты"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("ReserveWithdraw & ConfirmOperation"),
		allure.WithEpic("Управление счетами"),
		allure.WithFeature("Списание средств с реальным хранилищем"),
		allure.WithStory("Успешное резервирование средств с фиксацией операции и последующим подтверждением"),
	)
}
