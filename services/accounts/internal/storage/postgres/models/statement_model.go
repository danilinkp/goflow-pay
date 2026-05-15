package models

import (
	"accounts/internal/domain/entities"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type StatementModel struct {
	StatementId    uuid.UUID       `db:"statement_id"`
	AccountId      uuid.UUID       `db:"account_id"`
	CompanyId      uuid.UUID       `db:"company_id"`
	InitiatorId    uuid.UUID       `db:"initiator_id"`
	PeriodFrom     time.Time       `db:"period_from"`
	PeriodTo       time.Time       `db:"period_to"`
	OpeningBalance int64           `db:"opening_balance"`
	ClosingBalance int64           `db:"closing_balance"`
	TotalDebit     int64           `db:"total_debit"`
	TotalCredit    int64           `db:"total_credit"`
	Currency       string          `db:"currency"`
	Entries        json.RawMessage `db:"entries"`
	UpdatedAt      time.Time       `db:"updated_at"`
	CreatedAt      time.Time       `db:"created_at"`
}

func (m *StatementModel) ToDomain() *entities.Statement {
	var domainEntries []entities.StatementEntry

	if len(m.Entries) > 0 {
		if err := json.Unmarshal(m.Entries, &domainEntries); err != nil {
			return nil
		}
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
		domainEntries,
		m.UpdatedAt,
		m.CreatedAt,
	)
}

func ToStatementModel(statement *entities.Statement) *StatementModel {
	entriesJSON, _ := json.Marshal(statement.Entries())

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
		Entries:        entriesJSON,
		UpdatedAt:      statement.UpdatedAt(),
		CreatedAt:      statement.CreatedAt(),
	}
}

func StatementColumns() []string {
	return []string{
		"statement_id",
		"account_id",
		"company_id",
		"initiator_id",
		"period_from",
		"period_to",
		"opening_balance",
		"closing_balance",
		"total_debit",
		"total_credit",
		"currency",
		"entries",
		"updated_at",
		"created_at",
	}
}
