package request

import "github.com/google/uuid"

type TransferRequest struct {
	FromAccountId  uuid.UUID
	ToAccountId    uuid.UUID
	Amount         int64
	Currency       string
	IdempotencyKey string
}
