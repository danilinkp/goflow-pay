package models

import (
	"accounts/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type AccountOperationModel struct {
	OperationId     uuid.UUID `bson:"_id"`
	AccountId       uuid.UUID `bson:"account_id"`
	CounterpartyId  uuid.UUID `bson:"counterparty_id"`
	TransactionId   uuid.UUID `bson:"transaction_id"`
	OperationType   string    `bson:"operation_type"`
	OperationStatus string    `bson:"operation_status"`
	Amount          int64     `bson:"amount"`
	BalanceAfter    int64     `bson:"balance_after"`
	UpdatedAt       time.Time `bson:"updated_at"`
	CreatedAt       time.Time `bson:"created_at"`
}

func (m *AccountOperationModel) ToDomain() *entities.AccountOperation {
	return entities.ReconstructAccountOperation(
		m.OperationId,
		m.AccountId,
		m.CounterpartyId,
		m.TransactionId,
		entities.OperationType(m.OperationType),
		entities.OperationStatus(m.OperationStatus),
		m.Amount,
		m.BalanceAfter,
		m.CreatedAt,
		m.UpdatedAt,
	)
}

func ToAccountOperationModel(a *entities.AccountOperation) *AccountOperationModel {
	return &AccountOperationModel{
		OperationId:     a.AccountOperationId(),
		AccountId:       a.AccountId(),
		CounterpartyId:  a.CounterpartyId(),
		TransactionId:   a.TransactionId(),
		OperationType:   a.OperationType().String(),
		OperationStatus: a.OperationStatus().String(),
		Amount:          a.Amount(),
		BalanceAfter:    a.BalanceAfter(),
		CreatedAt:       a.CreatedAt(),
		UpdatedAt:       a.UpdatedAt(),
	}
}
