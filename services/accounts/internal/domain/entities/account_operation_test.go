package entities_test

import (
	"accounts/internal/domain/entities"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccountOperation_Success(t *testing.T) {
	accountId := uuid.New()
	transactionId := uuid.New()

	op, err := entities.NewAccountOperation(accountId, transactionId, entities.Deposit, entities.PendingStatus, 100)

	require.NoError(t, err)
	assert.Equal(t, accountId, op.AccountId())
	assert.Equal(t, transactionId, op.TransactionId())
	assert.Equal(t, entities.Deposit, op.OperationType())
	assert.Equal(t, entities.PendingStatus, op.OperationStatus())
	assert.Equal(t, int64(100), op.Amount())
	assert.NotEqual(t, uuid.Nil, op.AccountOperationId())
}

func TestNewAccountOperation_NilAccountId(t *testing.T) {
	_, err := entities.NewAccountOperation(uuid.Nil, uuid.New(), entities.Deposit, entities.PendingStatus, 100)
	assert.Error(t, err)
}

func TestNewAccountOperation_NilTransactionId(t *testing.T) {
	_, err := entities.NewAccountOperation(uuid.New(), uuid.Nil, entities.Deposit, entities.PendingStatus, 100)
	assert.Error(t, err)
}

func TestNewAccountOperation_InvalidOperationType(t *testing.T) {
	_, err := entities.NewAccountOperation(uuid.New(), uuid.New(), "invalid", entities.PendingStatus, 100)
	assert.Error(t, err)
}

func TestNewAccountOperation_InvalidOperationStatus(t *testing.T) {
	_, err := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, "invalid", 100)
	assert.Error(t, err)
}

func TestNewAccountOperation_ZeroAmount(t *testing.T) {
	_, err := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, 0)
	assert.Error(t, err)
}

func TestNewAccountOperation_NegativeAmount(t *testing.T) {
	_, err := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, -1)
	assert.Error(t, err)
}

func TestAccountOperation_UpdateOperationStatus_Success(t *testing.T) {
	op, _ := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, 100)

	err := op.UpdateOperationStatus(entities.SuccessStatus)

	require.NoError(t, err)
	assert.Equal(t, entities.SuccessStatus, op.OperationStatus())
}

func TestAccountOperation_UpdateOperationStatus_InvalidStatus(t *testing.T) {
	op, _ := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, 100)

	err := op.UpdateOperationStatus("invalid")

	assert.Error(t, err)
	assert.Equal(t, entities.PendingStatus, op.OperationStatus())
}

func TestAccountOperation_UpdateOperationStatus_AlreadyCompleted(t *testing.T) {
	op, _ := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, 100)
	_ = op.UpdateOperationStatus(entities.SuccessStatus)

	err := op.UpdateOperationStatus(entities.FailedStatus)

	assert.Error(t, err)
	assert.Equal(t, entities.SuccessStatus, op.OperationStatus())
}

func TestAccountOperation_UpdateOperationStatus_AlreadyFailed(t *testing.T) {
	op, _ := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, 100)
	_ = op.UpdateOperationStatus(entities.FailedStatus)

	err := op.UpdateOperationStatus(entities.SuccessStatus)

	assert.Error(t, err)
}

func TestAccountOperation_UpdateAmount_Success(t *testing.T) {
	op, _ := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, 100)

	err := op.UpdateAmount(200)

	require.NoError(t, err)
	assert.Equal(t, int64(200), op.Amount())
}

func TestAccountOperation_UpdateAmount_Zero(t *testing.T) {
	op, _ := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, 100)

	err := op.UpdateAmount(0)

	assert.Error(t, err)
	assert.Equal(t, int64(100), op.Amount())
}

func TestAccountOperation_UpdateAmount_Negative(t *testing.T) {
	op, _ := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, 100)

	err := op.UpdateAmount(-50)

	assert.Error(t, err)
	assert.Equal(t, int64(100), op.Amount())
}

func TestAccountOperation_ToOutboxEvent_Pending_ReturnsNil(t *testing.T) {
	op, _ := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, 100)

	evt, err := op.ToOutboxEvent()

	require.NoError(t, err)
	assert.Nil(t, evt)
}

func TestAccountOperation_ToOutboxEvent_Success(t *testing.T) {
	op, _ := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Deposit, entities.PendingStatus, 100)
	_ = op.UpdateOperationStatus(entities.SuccessStatus)

	evt, err := op.ToOutboxEvent()

	require.NoError(t, err)
	require.NotNil(t, evt)
	assert.NotEqual(t, uuid.Nil, evt.ID)
	assert.NotEmpty(t, evt.Payload)
}

func TestAccountOperation_ToOutboxEvent_Failed(t *testing.T) {
	op, _ := entities.NewAccountOperation(uuid.New(), uuid.New(), entities.Withdrawal, entities.PendingStatus, 100)
	_ = op.UpdateOperationStatus(entities.FailedStatus)

	evt, err := op.ToOutboxEvent()

	require.NoError(t, err)
	require.NotNil(t, evt)
}
