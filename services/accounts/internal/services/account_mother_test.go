package services_test

import (
	"accounts/internal/domain/entities"
	"accounts/internal/services"

	"github.com/google/uuid"
)

type AccountMother struct{}

func (AccountMother) ValidActiveAccount() *entities.Account {
	return NewAccountBuilder().Build()
}

func (AccountMother) ZeroBalanceAccount() *entities.Account {
	return NewAccountBuilder().WithBalance(0).Build()
}

func (AccountMother) InactiveAccount() *entities.Account {
	return NewAccountBuilder().WithStatus(entities.InactiveStatus).Build()
}

func (AccountMother) ValidBankOperationInput(accountID, bankAccountID uuid.UUID) *services.BankOperationInput {
	return NewBankOperationInputBuilder().
		WithAccountID(accountID).
		WithBankAccountID(bankAccountID).
		Build()
}
