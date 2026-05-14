package entities_test

import (
	"accounts/internal/domain/entities"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBankOperation_Success(t *testing.T) {
	accountId := uuid.New()
	bankAccountId := uuid.New()
	initiatorId := uuid.New()

	bo, err := entities.NewBankOperation(
		accountId, bankAccountId, initiatorId,
		"SberBank",
		entities.Deposit, entities.PendingStatus,
		500, 1200,
		"key-123", "",
	)

	require.NoError(t, err)
	assert.Equal(t, accountId, bo.AccountId())
	assert.Equal(t, bankAccountId, bo.BankAccountId())
	assert.Equal(t, initiatorId, bo.InitiatorId())
	assert.Equal(t, "SberBank", bo.BankName())
	assert.Equal(t, entities.Deposit, bo.OperationType())
	assert.Equal(t, entities.PendingStatus, bo.OperationStatus())
	assert.Equal(t, int64(500), bo.Amount())
	assert.Equal(t, int64(1200), bo.BalanceAfter())
	assert.Equal(t, "key-123", bo.IdempotencyKey())
	assert.NotEqual(t, uuid.Nil, bo.BankOperationId())
}

func TestNewBankOperation_NilAccountId(t *testing.T) {
	_, err := entities.NewBankOperation(
		uuid.Nil, uuid.New(), uuid.New(),
		"Bank",
		entities.Deposit, entities.PendingStatus,
		500, 0, "key", "",
	)
	assert.Error(t, err)
}

func TestNewBankOperation_NilBankAccountId(t *testing.T) {
	_, err := entities.NewBankOperation(
		uuid.New(), uuid.Nil, uuid.New(),
		"Bank",
		entities.Deposit, entities.PendingStatus,
		500, 0, "key", "",
	)
	assert.Error(t, err)
}

func TestNewBankOperation_NilInitiatorId(t *testing.T) {
	_, err := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.Nil,
		"Bank",
		entities.Deposit, entities.PendingStatus,
		500, 0, "key", "",
	)
	assert.Error(t, err)
}

func TestNewBankOperation_EmptyBankName(t *testing.T) {
	_, err := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.New(),
		"",
		entities.Deposit, entities.PendingStatus,
		500, 0, "key", "",
	)
	assert.Error(t, err)
}

func TestNewBankOperation_InvalidOperationType(t *testing.T) {
	_, err := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.New(),
		"Bank",
		entities.OperationType("invalid"), entities.PendingStatus,
		500, 0, "key", "",
	)
	assert.Error(t, err)
}

func TestNewBankOperation_InvalidOperationStatus(t *testing.T) {
	_, err := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.New(),
		"Bank",
		entities.Deposit, entities.OperationStatus("invalid"),
		500, 0, "key", "",
	)
	assert.Error(t, err)
}

func TestNewBankOperation_ZeroAmount(t *testing.T) {
	_, err := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.New(),
		"Bank",
		entities.Deposit, entities.PendingStatus,
		0, 0, "key", "",
	)
	assert.Error(t, err)
}

func TestNewBankOperation_NegativeAmount(t *testing.T) {
	_, err := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.New(),
		"Bank",
		entities.Deposit, entities.PendingStatus,
		-100, 0, "key", "",
	)
	assert.Error(t, err)
}

func TestNewBankOperation_EmptyIdempotencyKey(t *testing.T) {
	_, err := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.New(),
		"Bank",
		entities.Deposit, entities.PendingStatus,
		500, 0, "", "",
	)
	assert.Error(t, err)
}

func TestBankOperation_UpdateOperationStatus_Success(t *testing.T) {
	bo := newTestBankOperation(t)

	err := bo.UpdateOperationStatus(entities.SuccessStatus)

	require.NoError(t, err)
	assert.Equal(t, entities.SuccessStatus, bo.OperationStatus())
}

func TestBankOperation_UpdateOperationStatus_InvalidStatus(t *testing.T) {
	bo := newTestBankOperation(t)

	err := bo.UpdateOperationStatus("invalid")

	assert.Error(t, err)
	assert.Equal(t, entities.PendingStatus, bo.OperationStatus())
}

func TestBankOperation_UpdateOperationStatus_AlreadyCompleted(t *testing.T) {
	bo := newTestBankOperation(t)
	_ = bo.UpdateOperationStatus(entities.SuccessStatus)

	err := bo.UpdateOperationStatus(entities.FailedStatus)

	assert.Error(t, err)
	assert.Equal(t, entities.SuccessStatus, bo.OperationStatus())
}

func TestBankOperation_UpdateOperationStatus_AlreadyFailed(t *testing.T) {
	bo := newTestBankOperation(t)
	_ = bo.UpdateOperationStatus(entities.FailedStatus)

	err := bo.UpdateOperationStatus(entities.SuccessStatus)

	assert.Error(t, err)
	assert.Equal(t, entities.FailedStatus, bo.OperationStatus())
}

func TestBankOperation_UpdateExternalId_Success(t *testing.T) {
	bo := newTestBankOperation(t)

	err := bo.UpdateExternalId("ext-123")

	require.NoError(t, err)
	assert.Equal(t, "ext-123", bo.ExternalId())
}

func TestBankOperation_UpdateExternalId_SameValue(t *testing.T) {
	bo, _ := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.New(),
		"Bank",
		entities.Deposit, entities.PendingStatus,
		500, 0, "key", "same",
	)

	err := bo.UpdateExternalId("same")

	assert.Error(t, err)
	assert.Equal(t, "same", bo.ExternalId())
}

func TestBankOperation_UpdateAmount_Success(t *testing.T) {
	bo := newTestBankOperation(t)

	err := bo.UpdateAmount(1000)

	require.NoError(t, err)
	assert.Equal(t, int64(1000), bo.Amount())
}

func TestBankOperation_UpdateAmount_Zero(t *testing.T) {
	bo := newTestBankOperation(t)

	err := bo.UpdateAmount(0)

	assert.Error(t, err)
	assert.Equal(t, int64(500), bo.Amount())
}

func TestBankOperation_UpdateAmount_Negative(t *testing.T) {
	bo := newTestBankOperation(t)

	err := bo.UpdateAmount(-100)

	assert.Error(t, err)
	assert.Equal(t, int64(500), bo.Amount())
}

func TestBankOperation_ToOutboxEvent_Pending_ReturnsNil(t *testing.T) {
	bo := newTestBankOperation(t)

	evt, err := bo.ToOutboxEvent()

	require.NoError(t, err)
	assert.Nil(t, evt)
}

func TestBankOperation_ToOutboxEvent_DepositSuccess(t *testing.T) {
	bo := newTestBankOperation(t)
	_ = bo.UpdateOperationStatus(entities.SuccessStatus)

	evt, err := bo.ToOutboxEvent()

	require.NoError(t, err)
	require.NotNil(t, evt)
	assert.NotEmpty(t, evt.Payload)
}

func TestBankOperation_ToOutboxEvent_DepositFailed(t *testing.T) {
	bo := newTestBankOperation(t)
	_ = bo.UpdateOperationStatus(entities.FailedStatus)

	evt, err := bo.ToOutboxEvent()

	require.NoError(t, err)
	require.NotNil(t, evt)
}

func TestBankOperation_ToOutboxEvent_WithdrawalSuccess(t *testing.T) {
	bo, _ := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.New(),
		"Bank",
		entities.Withdrawal, entities.PendingStatus,
		500, 0, "key", "",
	)
	_ = bo.UpdateOperationStatus(entities.SuccessStatus)

	evt, err := bo.ToOutboxEvent()

	require.NoError(t, err)
	require.NotNil(t, evt)
}

func TestBankOperation_ToOutboxEvent_WithdrawalFailed(t *testing.T) {
	bo, _ := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.New(),
		"Bank",
		entities.Withdrawal, entities.PendingStatus,
		500, 0, "key", "",
	)
	_ = bo.UpdateOperationStatus(entities.FailedStatus)

	evt, err := bo.ToOutboxEvent()

	require.NoError(t, err)
	require.NotNil(t, evt)
}

func newTestBankOperation(t *testing.T) *entities.BankOperation {
	t.Helper()
	bo, err := entities.NewBankOperation(
		uuid.New(), uuid.New(), uuid.New(),
		"Bank",
		entities.Deposit, entities.PendingStatus,
		500, 1000, "key", "",
	)
	require.NoError(t, err)
	return bo
}
