package request

import "github.com/google/uuid"

type BankOperationRequest struct {
	AccountID      uuid.UUID
	BankAccountID  uuid.UUID
	Amount         int64
	IdempotencyKey string
}
