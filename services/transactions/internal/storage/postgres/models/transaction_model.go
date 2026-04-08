package models

import (
	"time"
	"transactions/internal/domain/entities"

	"github.com/google/uuid"
)

type TransactionModel struct {
	TransactionId  uuid.UUID `db:"transaction_id"`
	FromAccountId  uuid.UUID `db:"from_account_id"`
	ToAccountId    uuid.UUID `db:"to_account_id"`
	Amount         int64     `db:"amount"`
	Currency       string    `db:"currency"`
	IdempotencyKey string    `db:"idempotency_key"`
	Status         string    `db:"status"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

func (m *TransactionModel) ToDomain() *entities.Transaction {
	return entities.ReconstructTransaction(
		m.TransactionId,
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
