package models

import (
	"time"
	"transactions/internal/domain/entities"

	"github.com/google/uuid"
)

type TransactionModel struct {
	TransactionId  uuid.UUID `bson:"_id"`
	InitiatorId    uuid.UUID `bson:"initiator_id"`
	FromAccountId  uuid.UUID `bson:"from_account_id"`
	ToAccountId    uuid.UUID `bson:"to_account_id"`
	Amount         int64     `bson:"amount"`
	Currency       string    `bson:"currency"`
	IdempotencyKey string    `bson:"idempotency_key"`
	Status         string    `bson:"status"`
	CreatedAt      time.Time `bson:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at"`
}

func (m *TransactionModel) ToDomain() *entities.Transaction {
	return entities.ReconstructTransaction(
		m.TransactionId,
		m.InitiatorId,
		m.FromAccountId,
		m.ToAccountId,
		m.Amount,
		entities.Currency(m.Currency),
		entities.TransactionStatus(m.Status),
		m.IdempotencyKey,
		m.CreatedAt,
		m.UpdatedAt,
	)
}

func ToTransactionModel(transaction *entities.Transaction) *TransactionModel {
	return &TransactionModel{
		TransactionId:  transaction.TransactionID(),
		InitiatorId:    transaction.InitiatorId(),
		FromAccountId:  transaction.FromAccountID(),
		ToAccountId:    transaction.ToAccountID(),
		Amount:         transaction.Amount(),
		Currency:       transaction.Currency().String(),
		IdempotencyKey: transaction.IdempotencyKey(),
		Status:         transaction.Status().String(),
		CreatedAt:      transaction.CreatedAt(),
		UpdatedAt:      transaction.UpdatedAt(),
	}
}
