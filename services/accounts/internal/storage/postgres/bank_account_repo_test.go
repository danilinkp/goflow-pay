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

func newBankAccountRepo() *postgres.BankAccountRepo {
	return postgres.NewBankAccountRepo(testPool, testGetter)
}

func TestBankAccountRepo_Save_And_GetById(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newBankAccountRepo()

	bankAcc, err := entities.NewBankAccount(uuid.New(), "Alfa Bank", "044525225", "40702840500000000001", "USD")
	require.NoError(t, err, "Сущность BankAccount должна создаваться без ошибок")
	require.NotNil(t, bankAcc)

	err = repo.Save(ctx, bankAcc)
	require.NoError(t, err)

	got, err := repo.GetById(ctx, bankAcc.BankAccountId())
	require.NoError(t, err)
	assert.Equal(t, bankAcc.BankAccountId(), got.BankAccountId())
	assert.Equal(t, "044525225", got.BIC())
}

func TestBankAccountRepo_GetByCompanyId(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newBankAccountRepo()
	companyId := uuid.New()

	acc1, err := entities.NewBankAccount(
		companyId,
		"bank 1",
		"044525225",
		"40702840500000000001",
		entities.USD,
	)
	require.NoError(t, err, "Первый счет должен быть валидным")
	require.NotNil(t, acc1)

	acc2, err := entities.NewBankAccount(
		companyId,
		"ВТБ",
		"044525593",
		"40802840511111111116",
		entities.USD,
	)
	require.NoError(t, err, "Второй счет должен быть валидным")
	require.NotNil(t, acc2)
	_ = repo.Save(ctx, acc1)
	_ = repo.Save(ctx, acc2)

	accounts, err := repo.GetByCompanyId(ctx, companyId)
	require.NoError(t, err)
	assert.Len(t, accounts, 2)
}
