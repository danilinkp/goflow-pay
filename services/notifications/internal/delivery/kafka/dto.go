package notificationskafka

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TransferPayload struct {
	InitiatorID   uuid.UUID `json:"initiator_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
}

type BankOpPayload struct {
	InitiatorID uuid.UUID `json:"initiator_id"`
	AccountID   uuid.UUID `json:"account_id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
}

type StatementPayload struct {
	InitiatorID    uuid.UUID       `json:"initiator_id"`
	AccountID      uuid.UUID       `json:"account_id"`
	PeriodFrom     time.Time       `json:"period_from"`
	PeriodTo       time.Time       `json:"period_to"`
	OpeningBalance int64           `json:"opening_balance"`
	ClosingBalance int64           `json:"closing_balance"`
	TotalDebit     int64           `json:"total_debit"`
	TotalCredit    int64           `json:"total_credit"`
	Currency       string          `json:"currency"`
	Entries        json.RawMessage `json:"entries"`
}
