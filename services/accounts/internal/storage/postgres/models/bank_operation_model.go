package models

import (
	"accounts/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type BankOperationModel struct {
	BankOperationId uuid.UUID `db:"bank_operation_id"`
	AccountId       uuid.UUID `db:"account_id"`
	BankAccountId   uuid.UUID `db:"bank_account_id"`
	InitiatorId     uuid.UUID `db:"initiator_id"`
	OperationType   string    `db:"operation_type"`
	OperationStatus string    `db:"operation_status"`
	Amount          int64     `db:"amount"`
	IdempotencyKey  string    `db:"idempotency_key"`
	ExternalId      string    `db:"external_id"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

func (m *BankOperationModel) ToDomain() *entities.BankOperation {
	return entities.ReconstructBankAccountOperation(
		m.BankOperationId,
		m.AccountId,
		m.BankAccountId,
		m.InitiatorId,
		entities.OperationType(m.OperationType),
		entities.OperationStatus(m.OperationStatus),
		m.Amount,
		m.IdempotencyKey,
		m.ExternalId,
		m.CreatedAt,
		m.UpdatedAt,
	)
}

func ToBankOperationModel(bankOperation *entities.BankOperation) *BankOperationModel {
	return &BankOperationModel{
		BankOperationId: bankOperation.BankOperationId(),
		AccountId:       bankOperation.AccountId(),
		BankAccountId:   bankOperation.BankAccountId(),
		InitiatorId:     bankOperation.InitiatorId(),
		OperationType:   bankOperation.OperationType().String(),
		OperationStatus: bankOperation.OperationStatus().String(),
		Amount:          bankOperation.Amount(),
		IdempotencyKey:  bankOperation.IdempotencyKey(),
		ExternalId:      bankOperation.ExternalId(),
		CreatedAt:       bankOperation.CreatedAt(),
		UpdatedAt:       bankOperation.UpdatedAt(),
	}
}

func BankOperationColumns() []string {
	return []string{
		"bank_operation_id",
		"account_id",
		"bank_account_id",
		"initiator_id",
		"operation_type",
		"operation_status",
		"amount",
		"idempotency_key",
		"external_id",
		"updated_at",
		"created_at",
	}
}
