package service_test

import (
	"transactions/internal/domain/entities"
	"transactions/internal/service"

	"github.com/google/uuid"
)

type TransferInputBuilder struct {
	initiatorID    uuid.UUID
	fromAccountID  uuid.UUID
	toAccountID    uuid.UUID
	amount         int64
	currency       entities.Currency
	idempotencyKey string
}

func NewTransferInputBuilder() *TransferInputBuilder {
	return &TransferInputBuilder{
		initiatorID:    uuid.New(),
		fromAccountID:  uuid.New(),
		toAccountID:    uuid.New(),
		amount:         1000,
		currency:       entities.USD,
		idempotencyKey: uuid.New().String(),
	}
}

func (b *TransferInputBuilder) WithInitiatorID(id uuid.UUID) *TransferInputBuilder {
	b.initiatorID = id
	return b
}

func (b *TransferInputBuilder) WithFromAccountID(id uuid.UUID) *TransferInputBuilder {
	b.fromAccountID = id
	return b
}

func (b *TransferInputBuilder) WithToAccountID(id uuid.UUID) *TransferInputBuilder {
	b.toAccountID = id
	return b
}

func (b *TransferInputBuilder) WithAmount(amount int64) *TransferInputBuilder {
	b.amount = amount
	return b
}

func (b *TransferInputBuilder) WithCurrency(currency entities.Currency) *TransferInputBuilder {
	b.currency = currency
	return b
}

func (b *TransferInputBuilder) WithIdempotencyKey(key string) *TransferInputBuilder {
	b.idempotencyKey = key
	return b
}

func (b *TransferInputBuilder) Build() *service.TransferInput {
	return &service.TransferInput{
		InitiatorID:    b.initiatorID,
		FromAccountId:  b.fromAccountID,
		ToAccountId:    b.toAccountID,
		Amount:         b.amount,
		Currency:       b.currency,
		IdempotencyKey: b.idempotencyKey,
	}
}
