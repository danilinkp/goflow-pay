package entities_test

import (
	"testing"
	"transactions/internal/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransaction_Success(t *testing.T) {
	from := uuid.New()
	to := uuid.New()

	tx, err := entities.NewTransaction(from, to, 1000, "USD", entities.PendingStatus, "key-123")

	require.NoError(t, err)
	assert.Equal(t, from, tx.FromAccountID())
	assert.Equal(t, to, tx.ToAccountID())
	assert.Equal(t, int64(1000), tx.Amount())
	assert.Equal(t, "USD", tx.Currency())
	assert.Equal(t, entities.PendingStatus, tx.Status())
}

func TestNewTransaction_NilFromAccount(t *testing.T) {
	_, err := entities.NewTransaction(uuid.Nil, uuid.New(), 1000, "USD", entities.PendingStatus, "key")
	assert.Error(t, err)
}

func TestNewTransaction_NilToAccount(t *testing.T) {
	_, err := entities.NewTransaction(uuid.New(), uuid.Nil, 1000, "USD", entities.PendingStatus, "key")
	assert.Error(t, err)
}

func TestNewTransaction_SameAccounts(t *testing.T) {
	id := uuid.New()
	_, err := entities.NewTransaction(id, id, 1000, "USD", entities.PendingStatus, "key")
	assert.Error(t, err)
}

func TestNewTransaction_ZeroAmount(t *testing.T) {
	_, err := entities.NewTransaction(uuid.New(), uuid.New(), 0, "USD", entities.PendingStatus, "key")
	assert.Error(t, err)
}

func TestNewTransaction_NegativeAmount(t *testing.T) {
	_, err := entities.NewTransaction(uuid.New(), uuid.New(), -100, "USD", entities.PendingStatus, "key")
	assert.Error(t, err)
}

func TestNewTransaction_EmptyCurrency(t *testing.T) {
	_, err := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "", entities.PendingStatus, "key")
	assert.Error(t, err)
}

func TestNewTransaction_InvalidStatus(t *testing.T) {
	_, err := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", "invalid", "key")
	assert.Error(t, err)
}

func TestNewTransaction_EmptyIdempotencyKey(t *testing.T) {
	_, err := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "")
	assert.Error(t, err)
}

func TestTransaction_UpdateStatus_Success(t *testing.T) {
	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "key")

	err := tx.UpdateStatus(entities.ProcessingStatus)

	require.NoError(t, err)
	assert.Equal(t, entities.ProcessingStatus, tx.Status())
}

func TestTransaction_UpdateStatus_InvalidStatus(t *testing.T) {
	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "key")

	err := tx.UpdateStatus("invalid")

	assert.Error(t, err)
	assert.Equal(t, entities.PendingStatus, tx.Status())
}

func TestTransaction_UpdateStatus_AlreadySuccess(t *testing.T) {
	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "key")
	_ = tx.UpdateStatus(entities.ProcessingStatus)
	_ = tx.UpdateStatus(entities.SuccessStatus)

	err := tx.UpdateStatus(entities.FailedStatus)

	assert.Error(t, err)
	assert.Equal(t, entities.SuccessStatus, tx.Status())
}

func TestTransaction_UpdateStatus_AlreadyFailed(t *testing.T) {
	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "key")
	_ = tx.UpdateStatus(entities.FailedStatus)

	err := tx.UpdateStatus(entities.SuccessStatus)

	assert.Error(t, err)
}

func TestTransaction_UpdateCurrency_Success(t *testing.T) {
	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "key")

	err := tx.UpdateCurrency("EUR")

	require.NoError(t, err)
	assert.Equal(t, "EUR", tx.Currency())
}

func TestTransaction_UpdateCurrency_Empty(t *testing.T) {
	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "key")

	err := tx.UpdateCurrency("")

	assert.Error(t, err)
	assert.Equal(t, "USD", tx.Currency())
}

func TestTransaction_ToOutboxEvent_Pending_ReturnsNil(t *testing.T) {
	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "key")

	evt, err := tx.ToOutboxEvent()

	require.NoError(t, err)
	assert.Nil(t, evt)
}

func TestTransaction_ToOutboxEvent_Processing_ReturnsNil(t *testing.T) {
	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "key")
	_ = tx.UpdateStatus(entities.ProcessingStatus)

	evt, err := tx.ToOutboxEvent()

	require.NoError(t, err)
	assert.Nil(t, evt)
}

func TestTransaction_ToOutboxEvent_Success(t *testing.T) {
	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "key")
	_ = tx.UpdateStatus(entities.ProcessingStatus)
	_ = tx.UpdateStatus(entities.SuccessStatus)

	evt, err := tx.ToOutboxEvent()

	require.NoError(t, err)
	require.NotNil(t, evt)
	assert.NotEmpty(t, evt.Payload)
	assert.NotEqual(t, uuid.Nil, evt.ID)
}

func TestTransaction_ToOutboxEvent_Failed(t *testing.T) {
	tx, _ := entities.NewTransaction(uuid.New(), uuid.New(), 1000, "USD", entities.PendingStatus, "key")
	_ = tx.UpdateStatus(entities.FailedStatus)

	evt, err := tx.ToOutboxEvent()

	require.NoError(t, err)
	require.NotNil(t, evt)
}
