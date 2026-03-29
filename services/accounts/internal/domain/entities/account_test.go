package entities_test

import (
	"accounts/internal/domain/entities"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccount_Success(t *testing.T) {
	companyId := uuid.New()

	acc, err := entities.NewAccount(companyId, 0, "USD", entities.ActiveStatus)

	require.NoError(t, err)
	assert.Equal(t, companyId, acc.CompanyId())
	assert.Equal(t, int64(0), acc.Balance())
	assert.Equal(t, "USD", acc.Currency())
	assert.Equal(t, entities.ActiveStatus, acc.Status())
	assert.NotEqual(t, uuid.Nil, acc.AccountId())
}

func TestNewAccount_NegativeBalance(t *testing.T) {
	_, err := entities.NewAccount(uuid.New(), -1, "USD", entities.ActiveStatus)

	assert.Error(t, err)
}

func TestNewAccount_EmptyCurrency(t *testing.T) {
	_, err := entities.NewAccount(uuid.New(), 0, "", entities.ActiveStatus)

	assert.Error(t, err)
}

func TestNewAccount_InvalidStatus(t *testing.T) {
	_, err := entities.NewAccount(uuid.New(), 0, "USD", entities.AccountStatus("invalid"))

	assert.Error(t, err)
}

func TestAccount_AddBalance_Success(t *testing.T) {
	acc, _ := entities.NewAccount(uuid.New(), 100, "USD", entities.ActiveStatus)

	err := acc.AddBalance(50)

	require.NoError(t, err)
	assert.Equal(t, int64(150), acc.Balance())
}

func TestAccount_AddBalance_Negative(t *testing.T) {
	acc, _ := entities.NewAccount(uuid.New(), 100, "USD", entities.ActiveStatus)

	err := acc.AddBalance(-10)

	assert.Error(t, err)
	assert.Equal(t, int64(100), acc.Balance())
}

func TestAccount_RemoveBalance_Success(t *testing.T) {
	acc, _ := entities.NewAccount(uuid.New(), 100, "USD", entities.ActiveStatus)

	err := acc.RemoveBalance(40)

	require.NoError(t, err)
	assert.Equal(t, int64(60), acc.Balance())
}

func TestAccount_RemoveBalance_Negative(t *testing.T) {
	acc, _ := entities.NewAccount(uuid.New(), 100, "USD", entities.ActiveStatus)

	err := acc.RemoveBalance(-10)

	assert.Error(t, err)
	assert.Equal(t, int64(100), acc.Balance())
}

func TestAccount_UpdateBalance_Success(t *testing.T) {
	acc, _ := entities.NewAccount(uuid.New(), 100, "USD", entities.ActiveStatus)

	err := acc.UpdateBalance(200)

	require.NoError(t, err)
	assert.Equal(t, int64(200), acc.Balance())
}

func TestAccount_UpdateBalance_Negative(t *testing.T) {
	acc, _ := entities.NewAccount(uuid.New(), 100, "USD", entities.ActiveStatus)

	err := acc.UpdateBalance(-1)

	assert.Error(t, err)
	assert.Equal(t, int64(100), acc.Balance())
}

func TestAccount_UpdateStatus_Success(t *testing.T) {
	acc, _ := entities.NewAccount(uuid.New(), 0, "USD", entities.ActiveStatus)

	err := acc.UpdateStatus(entities.InactiveStatus)

	require.NoError(t, err)
	assert.Equal(t, entities.InactiveStatus, acc.Status())
}

func TestAccount_UpdateStatus_Invalid(t *testing.T) {
	acc, _ := entities.NewAccount(uuid.New(), 0, "USD", entities.ActiveStatus)

	err := acc.UpdateStatus(entities.AccountStatus("invalid"))

	assert.Error(t, err)
	assert.Equal(t, entities.ActiveStatus, acc.Status())
}

func TestAccount_UpdateCurrency_Success(t *testing.T) {
	acc, _ := entities.NewAccount(uuid.New(), 0, "USD", entities.ActiveStatus)

	err := acc.UpdateCurrency("EUR")

	require.NoError(t, err)
	assert.Equal(t, "EUR", acc.Currency())
}

func TestAccount_UpdateCurrency_Empty(t *testing.T) {
	acc, _ := entities.NewAccount(uuid.New(), 0, "USD", entities.ActiveStatus)

	err := acc.UpdateCurrency("")

	assert.Error(t, err)
	assert.Equal(t, "USD", acc.Currency())
}
