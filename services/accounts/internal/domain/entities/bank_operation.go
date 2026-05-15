package entities

import (
	"encoding/json"
	"fmt"
	"shared/pkg/outbox"
	"time"

	"github.com/google/uuid"
)

type BankOperation struct {
	bankOperationId uuid.UUID
	accountId       uuid.UUID
	bankAccountId   uuid.UUID
	initiatorId     uuid.UUID
	bankName        string
	operationType   OperationType
	operationStatus OperationStatus
	amount          int64
	balanceAfter    int64
	idempotencyKey  string
	externalId      string
	createdAt       time.Time
	updatedAt       time.Time
}

func NewBankOperation(accountId, bankAccountId, initiatorId uuid.UUID, bankName string, operationType OperationType,
	operationStatus OperationStatus, amount, balanceAfter int64, idempotencyKey string, externalId string) (*BankOperation, error) {
	if accountId == uuid.Nil {
		return nil, fmt.Errorf("%s: accountId is required", "create bank operation")
	}
	if bankAccountId == uuid.Nil {
		return nil, fmt.Errorf("%s: bankAccountId is required", "create bank operation")
	}
	if initiatorId == uuid.Nil {
		return nil, fmt.Errorf("%s: initiatorId is required", "create bank operation")
	}
	if bankName == "" {
		return nil, fmt.Errorf("%s: bankName is required", "create bank operation")
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
	if idempotencyKey == "" {
		return nil, fmt.Errorf("%s: idempotency key is required", "create bank operation")
	}

	now := time.Now().UTC()
	return &BankOperation{
		bankOperationId: uuid.New(),
		accountId:       accountId,
		bankAccountId:   bankAccountId,
		initiatorId:     initiatorId,
		bankName:        bankName,
		operationType:   operationType,
		operationStatus: operationStatus,
		amount:          amount,
		balanceAfter:    balanceAfter,
		idempotencyKey:  idempotencyKey,
		externalId:      externalId,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

func ReconstructBankAccountOperation(bankOperationId, accountId, bankAccountId, initiatorId uuid.UUID, bankName string, operationType OperationType,
	status OperationStatus, amount, balanceAfter int64, idempotencyKey, externalId string, createdAt, updatedAt time.Time) *BankOperation {
	return &BankOperation{
		bankOperationId: bankOperationId,
		accountId:       accountId,
		bankAccountId:   bankAccountId,
		initiatorId:     initiatorId,
		bankName:        bankName,
		operationType:   operationType,
		operationStatus: status,
		amount:          amount,
		balanceAfter:    balanceAfter,
		idempotencyKey:  idempotencyKey,
		externalId:      externalId,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}

func (bo *BankOperation) BankOperationId() uuid.UUID       { return bo.bankOperationId }
func (bo *BankOperation) AccountId() uuid.UUID             { return bo.accountId }
func (bo *BankOperation) BankAccountId() uuid.UUID         { return bo.bankAccountId }
func (bo *BankOperation) InitiatorId() uuid.UUID           { return bo.initiatorId }
func (bo *BankOperation) BankName() string                 { return bo.bankName }
func (bo *BankOperation) OperationType() OperationType     { return bo.operationType }
func (bo *BankOperation) OperationStatus() OperationStatus { return bo.operationStatus }
func (bo *BankOperation) Amount() int64                    { return bo.amount }
func (bo *BankOperation) BalanceAfter() int64              { return bo.balanceAfter }
func (bo *BankOperation) IdempotencyKey() string           { return bo.idempotencyKey }
func (bo *BankOperation) ExternalId() string               { return bo.externalId }
func (bo *BankOperation) CreatedAt() time.Time             { return bo.createdAt }
func (bo *BankOperation) UpdatedAt() time.Time             { return bo.updatedAt }

func (bo *BankOperation) UpdateOperationStatus(newStatus OperationStatus) error {
	if !newStatus.IsValid() {
		return fmt.Errorf("%s: invalid operation status %v", "update bank operation", newStatus)
	}
	if bo.operationStatus == SuccessStatus || bo.operationStatus == FailedStatus {
		return fmt.Errorf("%s: cannot change status of completed operation", "update bank operation")
	}
	bo.operationStatus = newStatus
	bo.updatedAt = time.Now().UTC()
	return nil
}

func (bo *BankOperation) UpdateExternalId(newExternalId string) error {
	if bo.externalId == newExternalId {
		return fmt.Errorf("%s: cannot change externalId of completed operation", "update bank operation")
	}
	bo.externalId = newExternalId
	bo.updatedAt = time.Now().UTC()
	return nil
}

func (bo *BankOperation) UpdateAmount(newAmount int64) error {
	if newAmount <= 0 {
		return fmt.Errorf("%s: amount must be greater than zero", "update bank operation")
	}

	bo.amount = newAmount
	bo.updatedAt = time.Now().UTC()
	return nil
}

func (bo *BankOperation) ToOutboxEvent() (*outbox.Event, error) {
	if bo.operationStatus == PendingStatus {
		return nil, nil
	}

	payload, err := json.Marshal(struct {
		OperationID   uuid.UUID `json:"operation_id"`
		AccountID     uuid.UUID `json:"account_id"`
		BankAccountID uuid.UUID `json:"bank_account_id"`
		InitiatorId   uuid.UUID `json:"initiator_id"`
		BankName      string    `json:"bank_name"`
		Type          string    `json:"type"`
		Status        string    `json:"status"`
		Amount        int64     `json:"amount"`
		BalanceAfter  int64     `json:"balance_after"`
		ExternalId    string    `json:"external_id"`
		CreatedAt     time.Time `json:"created_at"`
	}{
		OperationID:   bo.bankOperationId,
		AccountID:     bo.accountId,
		BankAccountID: bo.bankAccountId,
		InitiatorId:   bo.initiatorId,
		BankName:      bo.bankName,
		Type:          string(bo.operationType),
		Status:        string(bo.operationStatus),
		Amount:        bo.amount,
		BalanceAfter:  bo.balanceAfter,
		ExternalId:    bo.externalId,
		CreatedAt:     bo.createdAt,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal account operation: %w", err)
	}

	var eventType outbox.EventType
	switch {
	case bo.operationType == Deposit && bo.operationStatus == SuccessStatus:
		eventType = outbox.EventBankDepositCompleted
	case bo.operationType == Deposit && bo.operationStatus == FailedStatus:
		eventType = outbox.EventBankDepositFailed
	case bo.operationType == Withdrawal && bo.operationStatus == SuccessStatus:
		eventType = outbox.EventBankWithdrawalCompleted
	case bo.operationType == Withdrawal && bo.operationStatus == FailedStatus:
		eventType = outbox.EventBankWithdrawalFailed
	}

	return outbox.WithType(&outbox.Event{
		AggregateID:   bo.accountId,
		AggregateType: "bank_operation",
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
	}, eventType), nil
}
