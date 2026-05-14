package models

import (
	"accounts/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type AccountOperationModel struct {
	OperationId     uuid.UUID `db:"operation_id"`
	AccountId       uuid.UUID `db:"account_id"`
	CounterpartyId  uuid.UUID `db:"counterparty_id"`
	TransactionId   uuid.UUID `db:"transaction_id"`
	OperationType   string    `db:"operation_type"`
	OperationStatus string    `db:"operation_status"`
	Amount          int64     `db:"amount"`
	BalanceAfter    int64     `db:"balance_after"`
	UpdatedAt       time.Time `db:"updated_at"`
	CreatedAt       time.Time `db:"created_at"`
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

func AccountOperationColumns() []string {
	return []string{
		"operation_id",
		"account_id",
		"counterparty_id",
		"transaction_id",
		"operation_type",
		"operation_status",
		"amount",
		"balance_after",
		"updated_at",
		"created_at",
	}
}
