package models

import (
	"accounts/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type BankOperationModel struct {
	BankOperationId uuid.UUID `bson:"_id"`
	AccountId       uuid.UUID `bson:"account_id"`
	BankAccountId   uuid.UUID `bson:"bank_account_id"`
	InitiatorId     uuid.UUID `bson:"initiator_id"`
	BankName        string    `bson:"bank_name"`
	OperationType   string    `bson:"operation_type"`
	OperationStatus string    `bson:"operation_status"`
	Amount          int64     `bson:"amount"`
	BalanceAfter    int64     `bson:"balance_after"`
	IdempotencyKey  string    `bson:"idempotency_key"`
	ExternalId      string    `bson:"external_id"`
	CreatedAt       time.Time `bson:"created_at"`
	UpdatedAt       time.Time `bson:"updated_at"`
}

func (m *BankOperationModel) ToDomain() *entities.BankOperation {
	return entities.ReconstructBankAccountOperation(
		m.BankOperationId,
		m.AccountId,
		m.BankAccountId,
		m.InitiatorId,
		m.BankName,
		entities.OperationType(m.OperationType),
		entities.OperationStatus(m.OperationStatus),
		m.Amount,
		m.BalanceAfter,
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
		BankName:        bankOperation.BankName(),
		OperationType:   bankOperation.OperationType().String(),
		OperationStatus: bankOperation.OperationStatus().String(),
		Amount:          bankOperation.Amount(),
		BalanceAfter:    bankOperation.BalanceAfter(),
		IdempotencyKey:  bankOperation.IdempotencyKey(),
		ExternalId:      bankOperation.ExternalId(),
		CreatedAt:       bankOperation.CreatedAt(),
		UpdatedAt:       bankOperation.UpdatedAt(),
	}
}
