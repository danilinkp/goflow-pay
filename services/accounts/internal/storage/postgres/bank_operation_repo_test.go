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

func newBankOperationRepo() *postgres.BankOperationRepo {
	return postgres.NewBankOperationRepo(testPool, testGetter)
}

func TestBankOperationRepo_Save_And_GetByIdempotencyKey(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newBankOperationRepo()
	key := "unique-key-123"
	userId := uuid.New()

	companyId := uuid.New()

	accRepo := postgres.NewAccountRepo(testPool, testGetter)
	acc, _ := entities.NewAccount(companyId, 1000, entities.USD, entities.ActiveStatus)
	require.NoError(t, accRepo.Save(ctx, acc))

	bankAccRepo := postgres.NewBankAccountRepo(testPool, testGetter)
	bankAcc, _ := entities.NewBankAccount(companyId, "Alfa Bank", "044525225", "40702840500000000001", "USD")
	require.NoError(t, bankAccRepo.Save(ctx, bankAcc))

	op, err := entities.NewBankOperation(acc.AccountId(), bankAcc.BankAccountId(), userId, "withdrawal", "pending", 1000, key, "")
	require.NoError(t, err)
	err = repo.Save(ctx, op)
	require.NoError(t, err)

	got, err := repo.GetByIdempotencyKey(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, op.BankOperationId(), got.BankOperationId())
}

func TestBankOperationRepo_UpdateStatusAndExternalID(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()

	companyId := uuid.New()

	accRepo := postgres.NewAccountRepo(testPool, testGetter)
	acc, _ := entities.NewAccount(companyId, 1000, entities.USD, entities.ActiveStatus)
	require.NoError(t, accRepo.Save(ctx, acc))

	bankAccRepo := postgres.NewBankAccountRepo(testPool, testGetter)
	bankAcc, _ := entities.NewBankAccount(companyId, "Alfa Bank", "044525225", "40702840500000000001", "USD")
	require.NoError(t, bankAccRepo.Save(ctx, bankAcc))

	repo := newBankOperationRepo()

	op, _ := entities.NewBankOperation(acc.AccountId(), bankAcc.BankAccountId(), uuid.New(), "deposit", "pending", 500, "key", "")
	_ = repo.Save(ctx, op)

	err := repo.UpdateStatusAndExternalID(ctx, op.BankOperationId(), "success", "ext-id-999")
	require.NoError(t, err)

	got, _ := repo.GetByIdempotencyKey(ctx, "key")
	assert.Equal(t, entities.SuccessStatus, got.OperationStatus())
	assert.Equal(t, "ext-id-999", got.ExternalId())
}
