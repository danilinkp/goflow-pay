package bank

import (
	"accounts/internal/domain/entities"
	"context"
	"errors"
	"math/rand/v2"

	"github.com/google/uuid"
)

type MockBankGateway struct {
	failureRate float64
}

func NewMockBankGateway(failureRate float64) *MockBankGateway {
	return &MockBankGateway{failureRate: failureRate}
}

func (m *MockBankGateway) Deposit(ctx context.Context, bankAccount *entities.BankAccount, amount int64, idempotencyKey string) (string, error) {
	if m.shouldFail() {
		return "", errors.New("bank gateway: deposit temporarily unavailable")
	}
	return "ext_" + uuid.New().String(), nil
}

func (m *MockBankGateway) Withdraw(ctx context.Context, bankAccount *entities.BankAccount, amount int64, idempotencyKey string) (string, error) {
	if m.shouldFail() {
		return "", errors.New("bank gateway: withdrawal temporarily unavailable")
	}
	return "ext_" + uuid.New().String(), nil
}

func (m *MockBankGateway) shouldFail() bool {
	return rand.Float64() < m.failureRate
}
