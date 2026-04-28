package service

import (
	"transactions/internal/domain/entities"

	"github.com/google/uuid"
)

type TransferInput struct {
	InitiatorID    uuid.UUID
	FromAccountId  uuid.UUID
	ToAccountId    uuid.UUID
	Amount         int64
	Currency       entities.Currency
	IdempotencyKey string
}
