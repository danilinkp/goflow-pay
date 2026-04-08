package request

import (
	"transactions/internal/domain/entities"

	"github.com/google/uuid"
)

type TransferRequest struct {
	FromAccountId  uuid.UUID
	ToAccountId    uuid.UUID
	Amount         int64
	Currency       entities.Currency
	IdempotencyKey string
}
