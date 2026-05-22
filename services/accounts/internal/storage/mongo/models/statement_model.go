package models

import (
	"accounts/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type StatementModel struct {
	StatementId    uuid.UUID             `bson:"_id"`
	AccountId      uuid.UUID             `bson:"account_id"`
	CompanyId      uuid.UUID             `bson:"company_id"`
	InitiatorId    uuid.UUID             `bson:"initiator_id"`
	PeriodFrom     time.Time             `bson:"period_from"`
	PeriodTo       time.Time             `bson:"period_to"`
	OpeningBalance int64                 `bson:"opening_balance"`
	ClosingBalance int64                 `bson:"closing_balance"`
	TotalDebit     int64                 `bson:"total_debit"`
	TotalCredit    int64                 `bson:"total_credit"`
	Currency       string                `bson:"currency"`
	Entries        []StatementEntryModel `bson:"entries"`
	UpdatedAt      time.Time             `bson:"updated_at"`
	CreatedAt      time.Time             `bson:"created_at"`
}

func (m *StatementModel) ToDomain() *entities.Statement {
	entries := make([]entities.StatementEntry, len(m.Entries))

	for i, entry := range m.Entries {
		entries[i] = entry.ToDomain()
	}

	return entities.ReconstructStatement(
		m.StatementId,
		m.AccountId,
		m.CompanyId,
		m.InitiatorId,
		m.PeriodFrom,
		m.PeriodTo,
		m.OpeningBalance,
		m.ClosingBalance,
		m.TotalDebit,
		m.TotalCredit,
		entities.Currency(m.Currency),
		entries,
		m.UpdatedAt,
		m.CreatedAt,
	)
}

func ToStatementModel(statement *entities.Statement) *StatementModel {
	entries := make([]StatementEntryModel, len(statement.Entries()))

	for i, entry := range statement.Entries() {
		entries[i] = ToStatementEntryModel(&entry)
	}

	return &StatementModel{
		StatementId:    statement.StatementId(),
		AccountId:      statement.AccountId(),
		CompanyId:      statement.CompanyId(),
		InitiatorId:    statement.InitiatorId(),
		PeriodFrom:     statement.PeriodFrom(),
		PeriodTo:       statement.PeriodTo(),
		OpeningBalance: statement.OpeningBalance(),
		ClosingBalance: statement.ClosingBalance(),
		TotalDebit:     statement.TotalDebit(),
		TotalCredit:    statement.TotalCredit(),
		Currency:       statement.Currency().String(),
		Entries:        entries,
		UpdatedAt:      statement.UpdatedAt(),
		CreatedAt:      statement.CreatedAt(),
	}
}
