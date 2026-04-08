package entities

import (
	"encoding/json"
	"fmt"
	"shared/outbox"
	"time"

	"github.com/google/uuid"
)

type TransactionStatus string

const (
	PendingStatus    TransactionStatus = "pending"
	SuccessStatus    TransactionStatus = "success"
	FailedStatus     TransactionStatus = "failed"
	ProcessingStatus TransactionStatus = "processing"
)

func (t TransactionStatus) String() string {
	return string(t)
}

func (s TransactionStatus) IsValid() bool {
	return s == PendingStatus || s == SuccessStatus || s == FailedStatus || s == ProcessingStatus
}

type Transaction struct {
	transactionId  uuid.UUID
	fromAccountId  uuid.UUID
	toAccountId    uuid.UUID
	amount         int64
	currency       Currency
	idempotencyKey string
	status         TransactionStatus
	createdAt      time.Time
	updatedAt      time.Time
}

func NewTransaction(fromAccountId uuid.UUID, toAccountId uuid.UUID, amount int64, currency Currency, status TransactionStatus, idempotencyKey string) (*Transaction, error) {
	if fromAccountId == uuid.Nil || toAccountId == uuid.Nil {
		return nil, fmt.Errorf("%s: accountId is required", "create transaction")
	}
	if fromAccountId == toAccountId {
		return nil, fmt.Errorf("%s: sender and receiver accounts must be different", "create transaction")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%s: amount must be positive", "create transaction")
	}
	if !currency.IsValid() {
		return nil, fmt.Errorf("%s: currency cannot be blank", "create transaction")
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("%s: invalid status: %v", "create transaction", status)
	}
	if idempotencyKey == "" {
		return nil, fmt.Errorf("%s: idempotency key is required", "create transaction")
	}

	now := time.Now().UTC()
	return &Transaction{
		transactionId:  uuid.New(),
		fromAccountId:  fromAccountId,
		toAccountId:    toAccountId,
		amount:         amount,
		currency:       currency,
		idempotencyKey: idempotencyKey,
		status:         status,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

func ReconstructTransaction(transactionId, fromAccountId, toAccountId uuid.UUID, amount int64, currency Currency, status TransactionStatus, idempotencyKey string, createdAt, updatedAt time.Time) *Transaction {
	return &Transaction{
		transactionId:  transactionId,
		fromAccountId:  fromAccountId,
		toAccountId:    toAccountId,
		amount:         amount,
		currency:       currency,
		idempotencyKey: idempotencyKey,
		status:         status,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

func (t *Transaction) TransactionID() uuid.UUID  { return t.transactionId }
func (t *Transaction) FromAccountID() uuid.UUID  { return t.fromAccountId }
func (t *Transaction) ToAccountID() uuid.UUID    { return t.toAccountId }
func (t *Transaction) Amount() int64             { return t.amount }
func (t *Transaction) Currency() Currency        { return t.currency }
func (t *Transaction) IdempotencyKey() string    { return t.idempotencyKey }
func (t *Transaction) Status() TransactionStatus { return t.status }
func (t *Transaction) CreatedAt() time.Time      { return t.createdAt }
func (t *Transaction) UpdatedAt() time.Time      { return t.updatedAt }

func (t *Transaction) UpdateCurrency(newCurrency Currency) error {
	if !newCurrency.IsValid() {
		return fmt.Errorf("%s: currency cannot be blank", "update currency")
	}
	t.currency = newCurrency
	t.updatedAt = time.Now().UTC()
	return nil
}

func (t *Transaction) UpdateStatus(newStatus TransactionStatus) error {
	if !newStatus.IsValid() {
		return fmt.Errorf("%s: invalid status: %v", "update status", newStatus)
	}
	if t.status == SuccessStatus || t.status == FailedStatus {
		return fmt.Errorf("%s: cannot change status of completed transaction", "update transaction status")
	}
	t.status = newStatus
	t.updatedAt = time.Now().UTC()
	return nil
}

func (t *Transaction) ToOutboxEvent() (*outbox.Event, error) {
	if t.status == PendingStatus || t.status == ProcessingStatus {
		return nil, nil
	}

	payload, err := json.Marshal(struct {
		TransactionID uuid.UUID `json:"transaction_id"`
		FromAccountID uuid.UUID `json:"from_account_id"`
		ToAccountID   uuid.UUID `json:"to_account_id"`
		Amount        int64     `json:"amount"`
		Currency      string    `json:"currency"`
		Status        string    `json:"status"`
		CreatedAt     time.Time `json:"created_at"`
	}{
		TransactionID: t.transactionId,
		FromAccountID: t.fromAccountId,
		ToAccountID:   t.toAccountId,
		Amount:        t.amount,
		Currency:      t.currency.String(),
		Status:        string(t.status),
		CreatedAt:     t.createdAt,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal transaction: %w", err)
	}

	var eventType outbox.EventType
	switch t.status {
	case SuccessStatus:
		eventType = outbox.EventTransferCompleted
	case FailedStatus:
		eventType = outbox.EventTransferFailed
	}

	return outbox.WithType(&outbox.Event{
		AggregateID:   t.transactionId,
		AggregateType: "transaction",
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
	}, eventType), nil
}
