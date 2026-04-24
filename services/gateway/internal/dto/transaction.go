package dto

import (
	"time"

	"github.com/google/uuid"
)

type TransferRequest struct {
	FromAccountId  uuid.UUID `json:"from_account_id"`
	ToAccountId    uuid.UUID `json:"to_account_id"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type TransactionResponse struct {
	TransactionId     uuid.UUID `json:"transaction_id"`
	InitiatorId       uuid.UUID `json:"initiator_id"`
	FromAccountId     uuid.UUID `json:"from_account_id"`
	ToAccountId       uuid.UUID `json:"to_account_id"`
	Amount            int64     `json:"amount"`
	Currency          string    `json:"currency"`
	IdempotencyKey    string    `json:"idempotency_key"`
	TransactionStatus string    `json:"transaction_status"`
	CreatedAt         time.Time `json:"created_at"`
}
