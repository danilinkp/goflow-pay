package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateAccountResponse struct {
	CompanyId uuid.UUID `json:"company_id"`
	Currency  string    `json:"currency"`
}

type LinkBankAccountRequest struct {
	AccountId         uuid.UUID `json:"account_id"`
	Name              string    `json:"name"`
	BIC               string    `json:"bic"`
	SettlementAccount string    `json:"settlement_account"`
	Currency          string    `json:"currency"`
}

type ReserveOperationRequest struct {
	AccountId     uuid.UUID `json:"account_id"`
	TransactionId uuid.UUID `json:"transaction_id"`
	Amount        int64     `json:"amount"`
}

type MakeBankOperationRequest struct {
	AccountId      uuid.UUID `json:"account_id"`
	BankAccountId  uuid.UUID `json:"bank_account_id"`
	Amount         int64     `json:"amount"`
	IdempotencyKey string    `json:"idempotency_key"`
}

// RESPONSES
type AccountResponse struct {
	AccountId uuid.UUID `json:"account_id"`
	CompanyId uuid.UUID `json:"company_id"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type BankAccountResponse struct {
	BankAccountId     uuid.UUID `json:"bank_account_id"`
	CompanyId         uuid.UUID `json:"company_id"`
	Name              string    `json:"name"`
	BIC               string    `json:"bic"`
	SettlementAccount string    `json:"settlement_account"`
	Currency          string    `json:"currency"`
	CreatedAt         time.Time `json:"created_at"`
}

type AccountOperationResponse struct {
	OperationId     uuid.UUID `json:"operation_id"`
	AccountId       uuid.UUID `json:"account_id"`
	TransactionId   uuid.UUID `json:"transaction_id"`
	OperationType   string    `json:"operation_type"`
	OperationStatus string    `json:"operation_status"`
	Amount          int64     `json:"amount"`
	CreatedAt       time.Time `json:"created_at"`
}

type BankOperationResponse struct {
	BankOperationId uuid.UUID `json:"bank_operation_id"`
	AccountId       uuid.UUID `json:"account_id"`
	BankAccountId   uuid.UUID `json:"bank_account_id"`
	InitiatorId     uuid.UUID `json:"initiator_id"`
	OperationType   string    `json:"operation_type"`
	OperationStatus string    `json:"operation_status"`
	Amount          int64     `json:"amount"`
	IdempotencyKey  string    `json:"idempotency_key"`
	ExternalId      string    `json:"external_id"`
	CreatedAt       time.Time `json:"created_at"`
}
