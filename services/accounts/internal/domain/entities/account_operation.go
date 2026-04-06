package entities

import (
	"encoding/json"
	"fmt"
	"shared/outbox"
	"time"

	"github.com/google/uuid"
)

type AccountOperation struct {
	operationId     uuid.UUID
	accountId       uuid.UUID
	transactionId   uuid.UUID
	operationType   OperationType
	operationStatus OperationStatus
	amount          int64
	createdAt       time.Time
	updatedAt       time.Time
}

func NewAccountOperation(accountId uuid.UUID, transactionId uuid.UUID, operationType OperationType, operationStatus OperationStatus, amount int64) (*AccountOperation, error) {
	if accountId == uuid.Nil {
		return nil, fmt.Errorf("%s: accountId is required", "create bank operation")
	}
	if transactionId == uuid.Nil {
		return nil, fmt.Errorf("%s: transactoin id is required", "create bank operation")
	}
	if !operationType.IsValid() {
		return nil, fmt.Errorf("%s: invalid operation type %v", "create bank operation", operationType)
	}
	if !operationStatus.IsValid() {
		return nil, fmt.Errorf("%s: invalid operation status %v", "create bank operation", operationStatus)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%s: amount must be greater than zero", "create bank operation")
	}

	now := time.Now().UTC()
	return &AccountOperation{
		operationId:     uuid.New(),
		accountId:       accountId,
		transactionId:   transactionId,
		operationType:   operationType,
		operationStatus: operationStatus,
		amount:          amount,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

func (ao *AccountOperation) AccountOperationId() uuid.UUID    { return ao.operationId }
func (ao *AccountOperation) AccountId() uuid.UUID             { return ao.accountId }
func (ao *AccountOperation) TransactionId() uuid.UUID         { return ao.transactionId }
func (ao *AccountOperation) OperationType() OperationType     { return ao.operationType }
func (ao *AccountOperation) OperationStatus() OperationStatus { return ao.operationStatus }
func (ao *AccountOperation) Amount() int64                    { return ao.amount }
func (ao *AccountOperation) CreatedAt() time.Time             { return ao.createdAt }
func (ao *AccountOperation) UpdatedAt() time.Time             { return ao.updatedAt }

func (ao *AccountOperation) UpdateOperationStatus(newStatus OperationStatus) error {
	if !newStatus.IsValid() {
		return fmt.Errorf("%s: invalid operation status %v", "update account operation", newStatus)
	}
	if ao.operationStatus == SuccessStatus || ao.operationStatus == FailedStatus {
		return fmt.Errorf("%s: cannot change status of completed operation", "update account operation")
	}
	ao.operationStatus = newStatus
	ao.updatedAt = time.Now().UTC()
	return nil
}

func (ao *AccountOperation) UpdateAmount(newAmount int64) error {
	if newAmount <= 0 {
		return fmt.Errorf("%s: amount must be greater than zero", "update account operation")
	}

	ao.amount = newAmount
	ao.updatedAt = time.Now().UTC()
	return nil
}

func (ao *AccountOperation) ToOutboxEvent() (*outbox.Event, error) {
	if ao.operationStatus == PendingStatus {
		return nil, nil
	}

	payload, err := json.Marshal(struct {
		OperationID   uuid.UUID `json:"operation_id"`
		AccountID     uuid.UUID `json:"account_id"`
		TransactionID uuid.UUID `json:"transaction_id"`
		Type          string    `json:"type"`
		Status        string    `json:"status"`
		Amount        int64     `json:"amount"`
		CreatedAt     time.Time `json:"created_at"`
	}{
		OperationID:   ao.operationId,
		AccountID:     ao.accountId,
		TransactionID: ao.transactionId,
		Type:          string(ao.operationType),
		Status:        string(ao.operationStatus),
		Amount:        ao.amount,
		CreatedAt:     ao.createdAt,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal account operation: %w", err)
	}

	var eventType outbox.EventType
	switch ao.operationStatus {
	case SuccessStatus:
		eventType = outbox.EventTransferCompleted
	case FailedStatus:
		eventType = outbox.EventTransferFailed
	}

	return outbox.WithType(&outbox.Event{
		AggregateID:   ao.accountId,
		AggregateType: "account_operation",
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
	}, eventType), nil
}
