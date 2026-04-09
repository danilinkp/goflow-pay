package postgres_test

import (
	"context"
	"testing"
	"time"
	"transactions/internal/domain/entities"
	"transactions/internal/storage/postgres"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTransactionRepo() *postgres.TransactionRepo {
	return postgres.NewTransactionRepo(testPool, testGetter)
}

func TestTransactionRepo_Save_And_Get(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newTransactionRepo()

	tx, err := entities.NewTransaction(
		uuid.New(),
		uuid.New(),
		1000,
		entities.RUB,
		entities.PendingStatus,
		"idempotency-key-1",
	)
	require.NoError(t, err)

	err = repo.Save(ctx, tx)
	require.NoError(t, err)

	got, err := repo.GetById(ctx, tx.TransactionID())
	require.NoError(t, err)
	assert.Equal(t, tx.IdempotencyKey(), got.IdempotencyKey())

	gotByKey, err := repo.GetByIdempotencyKey(ctx, "idempotency-key-1")
	require.NoError(t, err)
	assert.Equal(t, tx.TransactionID(), gotByKey.TransactionID())
}

func TestTransactionRepo_GetByAccountId(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newTransactionRepo()

	myAcc := uuid.New()
	otherAcc := uuid.New()

	tx1, _ := entities.NewTransaction(myAcc, otherAcc, 100, entities.RUB, "pending", "key-1")
	tx2, _ := entities.NewTransaction(otherAcc, myAcc, 200, entities.RUB, "pending", "key-2")

	_ = repo.Save(ctx, tx1)
	_ = repo.Save(ctx, tx2)

	txs, err := repo.GetByAccountId(ctx, myAcc)
	require.NoError(t, err)
	assert.Len(t, txs, 2)
}

func TestTransactionRepo_UpdateStatus(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newTransactionRepo()

	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 500, entities.RUB, "pending", "key-update")
	_ = repo.Save(ctx, tx)

	err := repo.UpdateStatus(ctx, tx.TransactionID(), "success")
	require.NoError(t, err)

	updated, _ := repo.GetById(ctx, tx.TransactionID())
	assert.Equal(t, entities.SuccessStatus, updated.Status())
}

func TestTransactionRepo_GetStale(t *testing.T) {
	t.Cleanup(func() { truncate(t) })
	ctx := context.Background()
	repo := newTransactionRepo()

	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 100, entities.RUB, "pending", "stale-key")
	_ = repo.Save(ctx, tx)

	_, _ = testPool.Exec(ctx, "UPDATE transactions SET updated_at = $1 WHERE idempotency_key = $2",
		time.Now().Add(-10*time.Minute), "stale-key")

	staleTxs, err := repo.GetStale(ctx, 5*time.Minute, []string{"pending"})
	require.NoError(t, err)

	assert.Len(t, staleTxs, 1)
	assert.Equal(t, "stale-key", staleTxs[0].IdempotencyKey())
}
