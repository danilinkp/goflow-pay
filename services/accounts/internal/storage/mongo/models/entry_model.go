package models

import (
	"accounts/internal/domain/entities"
	"time"
)

type StatementEntryModel struct {
	Date         time.Time `bson:"date"`
	EntryType    string    `bson:"entry_type"`
	Amount       int64     `bson:"amount"`
	BalanceAfter int64     `bson:"balance_after"`
	Counterparty string    `bson:"counterparty"`
}

func (s StatementEntryModel) ToDomain() entities.StatementEntry {
	return entities.StatementEntry{
		Date:         s.Date,
		EntryType:    entities.StatementEntryType(s.EntryType),
		Amount:       s.Amount,
		BalanceAfter: s.BalanceAfter,
		Counterparty: s.Counterparty,
	}
}

func ToStatementEntryModel(s *entities.StatementEntry) StatementEntryModel {
	return StatementEntryModel{
		Date:         s.Date,
		EntryType:    string(s.EntryType),
		Amount:       s.Amount,
		BalanceAfter: s.BalanceAfter,
		Counterparty: s.Counterparty,
	}
}
