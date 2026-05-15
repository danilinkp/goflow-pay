package postgres_test

import (
	"accounts/internal/domain/entities"
	"accounts/internal/storage/postgres"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStatementRepo() *postgres.StatementRepo {
	return postgres.NewStatementRepo(testPool, testGetter)
}

func TestStatementRepo_Save_And_GetByAccountIdAndPeriod(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newStatementRepo()

	accRepo := postgres.NewAccountRepo(testPool, testGetter)
	companyId := uuid.New()
	acc, _ := entities.NewAccount(companyId, 0, entities.USD, entities.ActiveStatus)
	require.NoError(t, accRepo.Save(ctx, acc))

	entries := []entities.StatementEntry{
		{
			Date:         time.Now().UTC().Truncate(time.Microsecond),
			EntryType:    entities.EntryTypeTransferIn,
			Amount:       1000,
			BalanceAfter: 1000,
			Counterparty: "John Doe",
		},
		{
			Date:         time.Now().Add(time.Minute).UTC().Truncate(time.Microsecond),
			EntryType:    entities.EntryTypeTransferOut,
			Amount:       500,
			BalanceAfter: 500,
			Counterparty: "Amazon",
		},
	}

	periodFrom := time.Now().Add(-time.Hour).UTC()
	periodTo := time.Now().Add(time.Hour).UTC()

	statement := entities.ReconstructStatement(
		uuid.New(),
		acc.AccountId(),
		companyId,
		uuid.New(),
		periodFrom,
		periodTo,
		0,
		500,
		1000,
		500,
		entities.USD,
		entries,
		time.Now(),
		time.Now(),
	)

	err := repo.Save(ctx, statement)
	require.NoError(t, err)

	got, err := repo.GetByAccountIdAndPeriod(ctx, acc.AccountId(), periodFrom.Add(-time.Minute), periodTo.Add(time.Minute))
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, statement.StatementId(), got.StatementId())
	assert.Equal(t, statement.AccountId(), got.AccountId())
	assert.Equal(t, len(statement.Entries()), len(got.Entries()))

	assert.Equal(t, statement.Entries()[0].Counterparty, got.Entries()[0].Counterparty)
	assert.Equal(t, statement.Entries()[1].Amount, got.Entries()[1].Amount)
}

func TestStatementRepo_GetByAccountIdAndPeriod_NotFound(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newStatementRepo()

	got, err := repo.GetByAccountIdAndPeriod(ctx, uuid.New(), time.Now(), time.Now().Add(time.Hour))

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestStatementRepo_DuplicateSave(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newStatementRepo()

	accRepo := postgres.NewAccountRepo(testPool, testGetter)
	acc, _ := entities.NewAccount(uuid.New(), 0, entities.USD, entities.ActiveStatus)
	_ = accRepo.Save(ctx, acc)

	id := uuid.New()
	statement := entities.ReconstructStatement(
		id, acc.AccountId(), uuid.New(), uuid.New(),
		time.Now(), time.Now().Add(time.Hour),
		0, 0, 0, 0, entities.USD, []entities.StatementEntry{}, time.Now(), time.Now(),
	)

	err := repo.Save(ctx, statement)
	require.NoError(t, err)

	err = repo.Save(ctx, statement)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}
