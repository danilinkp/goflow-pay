package postgres_test

import (
	"accounts/internal/domain/entities"
	"accounts/internal/storage/postgres"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAccountOperationRepo() *postgres.AccountOperationRepo {
	return postgres.NewAccountOperationRepo(testPool, testGetter)
}

func TestAccountOperationRepo_Save_And_GetByTransactionId(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newAccountOperationRepo()

	accRepo := postgres.NewAccountRepo(testPool, testGetter)
	acc, _ := entities.NewAccount(uuid.New(), 1000, entities.RUB, entities.ActiveStatus)
	counterparty, _ := entities.NewAccount(uuid.New(), 10000, entities.RUB, entities.ActiveStatus)
	require.NoError(t, accRepo.Save(ctx, counterparty))
	require.NoError(t, accRepo.Save(ctx, acc))

	txId := uuid.New()

	op1, _ := entities.NewAccountOperation(acc.AccountId(), counterparty.AccountId(), txId, entities.Deposit, "pending", 100, acc.Balance())
	op2, _ := entities.NewAccountOperation(acc.AccountId(), counterparty.AccountId(), txId, entities.Withdrawal, "pending", 100, acc.Balance())
	err := repo.Save(ctx, op1)
	require.NoError(t, err)
	_ = repo.Save(ctx, op2)

	ops, err := repo.GetByTransactionId(ctx, txId)
	require.NoError(t, err)
	assert.Len(t, ops, 2)
}

func TestAccountOperationRepo_UpdateStatus(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newAccountOperationRepo()

	accRepo := postgres.NewAccountRepo(testPool, testGetter)
	acc, _ := entities.NewAccount(uuid.New(), 1000, entities.RUB, entities.ActiveStatus)
	counterparty, _ := entities.NewAccount(uuid.New(), 10000, entities.RUB, entities.ActiveStatus)
	require.NoError(t, accRepo.Save(ctx, counterparty))
	require.NoError(t, accRepo.Save(ctx, acc))

	op, _ := entities.NewAccountOperation(acc.AccountId(), counterparty.AccountId(), uuid.New(), "deposit", "pending", 300, acc.Balance())
	err := repo.Save(ctx, op)
	require.NoError(t, err)

	err = repo.UpdateStatus(ctx, op.AccountOperationId(), "success")
	require.NoError(t, err)

	ops, _ := repo.GetByTransactionId(ctx, op.TransactionId())
	assert.Equal(t, entities.SuccessStatus, ops[0].OperationStatus())
}
