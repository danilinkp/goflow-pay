package services

import (
	"accounts/internal/domain/entities"

	"github.com/google/uuid"
)

type BankOperationInput struct {
	AccountID      uuid.UUID
	CompanyID      uuid.UUID
	BankAccountID  uuid.UUID
	InitiatorID    uuid.UUID
	Amount         int64
	IdempotencyKey string
}

type LinkBankInput struct {
	AccountID         uuid.UUID
	CompanyID         uuid.UUID
	Name              string
	Bic               string
	SettlementAccount string
	Currency          entities.Currency
}

type LinkBankOutput struct {
	AccountID     uuid.UUID
	BankAccountID uuid.UUID
	BankName      string
}
