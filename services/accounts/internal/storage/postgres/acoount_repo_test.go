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

func newAccountRepo() *postgres.AccountRepo {
	return postgres.NewAccountRepo(testPool, testGetter)
}

func newTestAccount(t *testing.T) *entities.Account {
	t.Helper()
	acc, err := entities.NewAccount(uuid.New(), 1000, entities.RUB, entities.ActiveStatus)
	require.NoError(t, err)
	return acc
}

func TestAccountRepo_Save_And_GetById(t *testing.T) {
	t.Cleanup(func() { truncate(t) })

	ctx := context.Background()
	repo := newAccountRepo()

	acc := newTestAccount(t)

	err := repo.Save(ctx, acc)
	require.NoError(t, err)

	got, err := repo.GetById(ctx, acc.AccountId())
	require.NoError(t, err)

	assert.Equal(t, acc.AccountId(), got.AccountId())
	assert.Equal(t, acc.CompanyId(), got.CompanyId())
	assert.Equal(t, acc.Balance(), got.Balance())
	assert.Equal(t, acc.Currency(), got.Currency())
	assert.Equal(t, acc.Status(), got.Status())
}

func TestAccountRepo_GetById_NotFound(t *testing.T) {
	t.Cleanup(func() { truncate(t) })

	ctx := context.Background()
	repo := newAccountRepo()

	_, err := repo.GetById(ctx, uuid.New())

	require.Error(t, err)
}

func TestAccountRepo_UpdateBalance(t *testing.T) {
	t.Cleanup(func() { truncate(t) })

	ctx := context.Background()
	repo := newAccountRepo()

	acc := newTestAccount(t)
	err := repo.Save(ctx, acc)
	require.NoError(t, err)

	err = repo.UpdateBalance(ctx, acc.AccountId(), 500)
	require.NoError(t, err)

	got, err := repo.GetById(ctx, acc.AccountId())
	require.NoError(t, err)

	assert.Equal(t, int64(1500), got.Balance())
}

func TestAccountRepo_UpdateBalance_NotFound(t *testing.T) {
	t.Cleanup(func() { truncate(t) })

	ctx := context.Background()
	repo := newAccountRepo()

	err := repo.UpdateBalance(ctx, uuid.New(), 500)
	require.Error(t, err)
}

func TestAccountRepo_GetByCompanyId(t *testing.T) {
	t.Cleanup(func() { truncate(t) })

	ctx := context.Background()
	repo := newAccountRepo()

	companyId := uuid.New()

	acc1, _ := entities.NewAccount(companyId, 100, entities.RUB, entities.ActiveStatus)
	acc2, _ := entities.NewAccount(companyId, 200, entities.RUB, entities.ActiveStatus)

	require.NoError(t, repo.Save(ctx, acc1))
	require.NoError(t, repo.Save(ctx, acc2))

	other, _ := entities.NewAccount(uuid.New(), 300, entities.RUB, entities.ActiveStatus)
	require.NoError(t, repo.Save(ctx, other))

	accounts, err := repo.GetByCompanyId(ctx, companyId)
	require.NoError(t, err)

	assert.Len(t, accounts, 2)
}

func TestAccountRepo_UpdateStatus(t *testing.T) {
	t.Cleanup(func() { truncate(t) })

	ctx := context.Background()
	repo := newAccountRepo()

	acc := newTestAccount(t)
	require.NoError(t, repo.Save(ctx, acc))

	err := repo.UpdateStatus(ctx, acc.AccountId(), entities.InactiveStatus)
	require.NoError(t, err)

	got, err := repo.GetById(ctx, acc.AccountId())
	require.NoError(t, err)

	assert.Equal(t, entities.InactiveStatus, got.Status())
}

func TestAccountRepo_ContextCancelled(t *testing.T) {
	t.Cleanup(func() { truncate(t) })

	ctx, cancel := context.WithCancel(context.Background())
	repo := newAccountRepo()
	acc := newTestAccount(t)

	cancel()

	err := repo.Save(ctx, acc)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), context.Canceled.Error())
}

func TestAccountRepo_DatabaseConnectionError(t *testing.T) {
	ctx := context.Background()

	repo := newAccountRepo()
	acc := newTestAccount(t)

	testPool.Close()

	err := repo.Save(ctx, acc)

	assert.Error(t, err)
}
