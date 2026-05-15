package entities

import "time"

type StatementEntryType string

const (
	EntryTypeTransferIn     StatementEntryType = "transfer_in"
	EntryTypeTransferOut    StatementEntryType = "transfer_out"
	EntryTypeBankDeposit    StatementEntryType = "bank_deposit"
	EntryTypeBankWithdrawal StatementEntryType = "bank_withdrawal"
)

type StatementEntry struct {
	Date         time.Time          `json:"date"`
	EntryType    StatementEntryType `json:"entry_type"`
	Amount       int64              `json:"amount"`
	BalanceAfter int64              `json:"balance_after"`
	Counterparty string             `json:"counterparty"`
}
