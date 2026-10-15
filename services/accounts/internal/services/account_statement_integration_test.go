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

func TestAccountService_GenerateStatement_Integration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx := context.Background()
		now := time.Now().UTC()
		svc, pool := setupIntegrationAccountService(t, func() time.Time { return now })

		companyID := uuid.New()
		initiatorID := uuid.New()

		targetAcc := NewAccountBuilder().
			WithCompanyID(companyID).
			WithBalance(10000).
			WithStatus(entities.ActiveStatus).
			WithCurrency(entities.RUB).
			Build()

		counterpartyAcc := NewAccountBuilder().
			WithCompanyID(uuid.New()).
			WithBalance(20000).
			WithStatus(entities.ActiveStatus).
			WithCurrency(entities.RUB).
			Build()

		emptyAcc := NewAccountBuilder().
			WithCompanyID(companyID).
			WithBalance(0).
			WithStatus(entities.ActiveStatus).
			WithCurrency(entities.RUB).
			Build()

		txID1 := uuid.New()
		txID2 := uuid.New()

		periodFrom := now.Add(-24 * time.Hour)
		periodTo := now.Add(24 * time.Hour)

		// Arrange (п. 17b методички: один раз в начале теста)
		a.Step("Arrange: подготовка счетов и генерация реальной цепочки операций в PostgreSQL", func(a *allure.Context) {
			insertTestAccount(t, ctx, pool, targetAcc)
			insertTestAccount(t, ctx, pool, counterpartyAcc)
			insertTestAccount(t, ctx, pool, emptyAcc)

			// 1. Операция списания (Withdrawal): 3 000 руб.
			op1, err := svc.ReserveWithdraw(ctx, targetAcc.AccountId(), counterpartyAcc.AccountId(), txID1, 3000)
			require.NoError(t, err)
			require.NotNil(t, op1)
			require.NoError(t, svc.ConfirmOperation(ctx, txID1))

			// 2. Операция зачисления (Deposit): 5 000 руб.
			op2, err := svc.ReserveDeposit(ctx, targetAcc.AccountId(), counterpartyAcc.AccountId(), txID2, 5000)
			require.NoError(t, err)
			require.NotNil(t, op2)
			require.NoError(t, svc.ConfirmOperation(ctx, txID2))
		})

		// Act 1: Формирование выписки по операциям за период
		var statement *entities.Statement
		var genErr error
		a.Step("Act 1: вызов GenerateStatement через AccountService", func(a *allure.Context) {
			statement, genErr = svc.GenerateStatement(ctx, targetAcc.AccountId(), initiatorID, companyID, periodFrom, periodTo)
		})

		// Assert 1: Проверка агрегатов и записей выписки
		a.Step("Assert 1: проверка корректности рассчитанных агрегатов выписки (Debit, Credit, Opening, Closing)", func(a *allure.Context) {
			require.NoError(t, genErr)
			require.NotNil(t, statement)

			assert.Equal(t, targetAcc.AccountId(), statement.AccountId())
			assert.Equal(t, companyID, statement.CompanyId())
			assert.Equal(t, initiatorID, statement.InitiatorId())
			assert.Equal(t, int64(10000), statement.OpeningBalance(), "начальный остаток должен быть 10 000 руб.")
			assert.Equal(t, int64(12000), statement.ClosingBalance(), "конечный остаток должен быть 12 000 руб. (10 000 - 3 000 + 5 000)")
			assert.Equal(t, int64(3000), statement.TotalCredit(), "кредитовый оборот (расход) должен быть 3 000 руб.")
			assert.Equal(t, int64(5000), statement.TotalDebit(), "дебетовый оборот (приход) должен быть 5 000 руб.")
			assert.Len(t, statement.Entries(), 2, "в выписке должно быть ровно 2 записи")

			// Проверка сохранения в таблице statements в PostgreSQL
			var dbStatementsCount int
			err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM statements WHERE statement_id = $1", statement.StatementId()).
				Scan(&dbStatementsCount)
			require.NoError(t, err)
			assert.Equal(t, 1, dbStatementsCount, "выписка должна сохраниться в PostgreSQL")

			// Проверка формирования Outbox-события в PostgreSQL (aggregate_id = account_id)
			var dbOutboxCount int
			err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox WHERE aggregate_id = $1 AND aggregate_type = 'statement'", targetAcc.AccountId()).
				Scan(&dbOutboxCount)
			require.NoError(t, err)
			assert.Equal(t, 1, dbOutboxCount, "в таблице outbox должно появиться событие генерации выписки")
		})

		// Act 2: Повторный запрос выписки (проверка идемпотентности и кэша БД)
		var cachedStatement *entities.Statement
		a.Step("Act 2: повторный вызов GenerateStatement за тот же период", func(a *allure.Context) {
			cachedStatement, genErr = svc.GenerateStatement(ctx, targetAcc.AccountId(), initiatorID, companyID, periodFrom, periodTo)
		})

		// Assert 2: Проверка возврата сохраненной выписки без дубликатов в БД
		a.Step("Assert 2: проверка возврата существующей выписки без повторной записи в PostgreSQL", func(a *allure.Context) {
			require.NoError(t, genErr)
			require.NotNil(t, cachedStatement)
			assert.Equal(t, statement.StatementId(), cachedStatement.StatementId())

			var count int
			err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM statements WHERE account_id = $1", targetAcc.AccountId()).
				Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count, "в БД должна остаться ровно 1 выписка за период")
		})

		// Act 3: Запрос выписки для счета без операций
		var emptyErr error
		a.Step("Act 3: попытка сгенерировать выписку для счета без операций", func(a *allure.Context) {
			_, emptyErr = svc.GenerateStatement(ctx, emptyAcc.AccountId(), initiatorID, companyID, periodFrom, periodTo)
		})

		// Assert 3: Проверка возврата доменной ошибки ErrNotAnyOperations
		a.Step("Assert 3: проверка ошибки ErrNotAnyOperations", func(a *allure.Context) {
			require.Error(t, emptyErr)
			assert.ErrorIs(t, emptyErr, services.ErrNotAnyOperations)
		})
	},
		allure.WithParentSuite("Интеграционные тесты"),
		allure.WithSuite("AccountService"),
		allure.WithSubSuite("GenerateStatement"),
		allure.WithEpic("Управление счетами"),
		allure.WithFeature("Банковская выписка"),
		allure.WithStory("Генерация выписки по реальным операциям в PostgreSQL, проверка агрегатов и Outbox"),
	)
}
