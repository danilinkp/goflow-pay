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

type GenerateStatementRequest struct {
	PeriodFrom time.Time `json:"period_from"`
	PeriodTo   time.Time `json:"period_to"`
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

type StatementEntry struct {
	Date         time.Time `json:"date"`
	EntryType    string    `json:"entry_type"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balance_after"`
	Counterparty string    `json:"counterparty"`
}

type StatementResponse struct {
	StatementId    uuid.UUID        `json:"statement_id"`
	AccountId      uuid.UUID        `json:"account_id"`
	CompanyId      uuid.UUID        `json:"company_id"`
	InitiatorId    uuid.UUID        `json:"initiator_id"`
	PeriodFrom     time.Time        `json:"period_from"`
	PeriodTo       time.Time        `json:"period_to"`
	OpeningBalance int64            `json:"opening_balance"`
	ClosingBalance int64            `json:"closing_balance"`
	TotalDebit     int64            `json:"total_debit"`
	TotalCredit    int64            `json:"total_credit"`
	Currency       string           `json:"currency"`
	Entries        []StatementEntry `json:"entries"`
}
