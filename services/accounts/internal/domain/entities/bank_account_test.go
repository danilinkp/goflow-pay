package entities_test

import (
	"accounts/internal/domain/entities"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBankAccount_Success(t *testing.T) {
	companyId := uuid.New()

	ba, err := entities.NewBankAccount(companyId, "Bank", "BIC123", "ACC123", "USD")

	require.NoError(t, err)
	assert.Equal(t, companyId, ba.CompanyId())
	assert.Equal(t, "Bank", ba.Name())
	assert.Equal(t, "BIC123", ba.BIC())
	assert.Equal(t, "ACC123", ba.SettlementAccount())
	assert.Equal(t, "USD", ba.Currency())
	assert.NotEqual(t, uuid.Nil, ba.BankAccountId())
}

func TestNewBankAccount_NilCompanyId(t *testing.T) {
	_, err := entities.NewBankAccount(uuid.Nil, "Bank", "BIC123", "ACC123", "USD")
	assert.Error(t, err)
}

func TestNewBankAccount_EmptyName(t *testing.T) {
	_, err := entities.NewBankAccount(uuid.New(), "", "BIC123", "ACC123", "USD")
	assert.Error(t, err)
}

func TestNewBankAccount_EmptyBic(t *testing.T) {
	_, err := entities.NewBankAccount(uuid.New(), "Bank", "", "ACC123", "USD")
	assert.Error(t, err)
}

func TestNewBankAccount_EmptySettlementAccount(t *testing.T) {
	_, err := entities.NewBankAccount(uuid.New(), "Bank", "BIC123", "", "USD")
	assert.Error(t, err)
}

func TestNewBankAccount_EmptyCurrency(t *testing.T) {
	_, err := entities.NewBankAccount(uuid.New(), "Bank", "BIC123", "ACC123", "")
	assert.Error(t, err)
}

func TestBankAccount_UpdateName_Success(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Old", "BIC", "ACC", "USD")

	err := ba.UpdateName("New")

	require.NoError(t, err)
	assert.Equal(t, "New", ba.Name())
}

func TestBankAccount_UpdateName_Empty(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Old", "BIC", "ACC", "USD")

	err := ba.UpdateName("")

	assert.Error(t, err)
	assert.Equal(t, "Old", ba.Name())
}

func TestBankAccount_UpdateBIC_Success(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Bank", "OldBIC", "ACC", "USD")

	err := ba.UpdateBIC("NewBIC")

	require.NoError(t, err)
	assert.Equal(t, "NewBIC", ba.BIC())
}

func TestBankAccount_UpdateBIC_Empty(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Bank", "OldBIC", "ACC", "USD")

	err := ba.UpdateBIC("")

	assert.Error(t, err)
	assert.Equal(t, "OldBIC", ba.BIC())
}

func TestBankAccount_UpdateSettlementAccount_Success(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Bank", "BIC", "OldACC", "USD")

	err := ba.UpdateSettlementAccount("NewACC")

	require.NoError(t, err)
	assert.Equal(t, "NewACC", ba.SettlementAccount())
}

func TestBankAccount_UpdateCurrency_Success(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Bank", "BIC", "ACC", "USD")

	err := ba.UpdateCurrency("EUR")

	require.NoError(t, err)
	assert.Equal(t, "EUR", ba.Currency())
}
