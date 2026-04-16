package notificationskafka

import "github.com/google/uuid"

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
