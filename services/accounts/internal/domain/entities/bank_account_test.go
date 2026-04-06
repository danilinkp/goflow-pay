package entities_test

import (
	"accounts/internal/domain/entities"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBIC        = "044525225"
	testAccountUSD = "40702840500000000001"
)

func TestNewBankAccount_Success(t *testing.T) {
	companyId := uuid.New()

	ba, err := entities.NewBankAccount(companyId, "Bank", testBIC, testAccountUSD, "USD")

	require.NoError(t, err)
	assert.Equal(t, companyId, ba.CompanyId())
	assert.Equal(t, "Bank", ba.Name())
	assert.Equal(t, testBIC, ba.BIC())
	assert.Equal(t, testAccountUSD, ba.SettlementAccount())
	assert.Equal(t, entities.Currency("USD"), ba.Currency())
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
	ba, _ := entities.NewBankAccount(uuid.New(), "Old", testBIC, testAccountUSD, "USD")

	err := ba.UpdateName("New")

	require.NoError(t, err)
	assert.Equal(t, "New", ba.Name())
}

func TestBankAccount_UpdateName_Empty(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Old", testBIC, testAccountUSD, "USD")

	err := ba.UpdateName("")

	assert.Error(t, err)
	assert.Equal(t, "Old", ba.Name())
}

func TestBankAccount_UpdateBIC_Success(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Bank", testBIC, testAccountUSD, "USD")

	err := ba.UpdateBIC("043525225")

	require.NoError(t, err)
	assert.Equal(t, "043525225", ba.BIC())
}

func TestBankAccount_UpdateBIC_Empty(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Bank", testBIC, testAccountUSD, "USD")

	err := ba.UpdateBIC("")

	assert.Error(t, err)
	assert.Equal(t, testBIC, ba.BIC())
}

func TestBankAccount_UpdateSettlementAccount_Success(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Bank", testBIC, testAccountUSD, "USD")

	err := ba.UpdateSettlementAccount("40702840500000000002")

	require.NoError(t, err)
	assert.Equal(t, "40702840500000000002", ba.SettlementAccount())
}

func TestBankAccount_UpdateCurrency_Success(t *testing.T) {
	ba, _ := entities.NewBankAccount(uuid.New(), "Bank", testBIC, testAccountUSD, "USD")

	err := ba.UpdateCurrency("EUR")

	require.Error(t, err)
	assert.Equal(t, entities.Currency("USD"), ba.Currency())
}
