package entities

import (
	"encoding/json"
	"fmt"
	"shared/pkg/outbox"
	"time"

	"github.com/google/uuid"
)

type Statement struct {
	statementId    uuid.UUID
	accountId      uuid.UUID
	companyId      uuid.UUID
	initiatorId    uuid.UUID
	periodFrom     time.Time
	periodTo       time.Time
	openingBalance int64
	closingBalance int64
	totalDebit     int64
	totalCredit    int64
	currency       Currency
	entries        []StatementEntry
	createdAt      time.Time
	updatedAt      time.Time
}

func NewStatement(accountId, companyId, initiatorId uuid.UUID,
	periodFrom, periodTo time.Time,
	openingBalance, closingBalance, totalDebit, totalCredit int64,
	currency Currency, entries []StatementEntry) (*Statement, error) {
	if accountId == uuid.Nil {
		return nil, fmt.Errorf("create statement: accountId is required")
	}
	if companyId == uuid.Nil {
		return nil, fmt.Errorf("create statement: companyId is required")
	}
	if initiatorId == uuid.Nil {
		return nil, fmt.Errorf("create statement: initiatorId is required")
	}
	if periodFrom.IsZero() {
		return nil, fmt.Errorf("create statement: periodFrom is required")
	}
	if periodTo.IsZero() {
		return nil, fmt.Errorf("create statement: periodTo is required")
	}
	if periodFrom.After(periodTo) {
		return nil, fmt.Errorf("create statement: periodFrom must be before periodTo")
	}
	if !currency.IsValid() {
		return nil, fmt.Errorf("create statement: currency is invalid")
	}

	now := time.Now().UTC()
	return &Statement{
		statementId:    uuid.New(),
		accountId:      accountId,
		companyId:      companyId,
		initiatorId:    initiatorId,
		periodFrom:     periodFrom,
		periodTo:       periodTo,
		openingBalance: openingBalance,
		closingBalance: closingBalance,
		totalDebit:     totalDebit,
		totalCredit:    totalCredit,
		currency:       currency,
		entries:        entries,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

func ReconstructStatement(
	id uuid.UUID,
	accountId uuid.UUID,
	companyId uuid.UUID,
	initiatorId uuid.UUID,
	periodFrom time.Time,
	periodTo time.Time,
	openingBalance int64,
	closingBalance int64,
	totalDebit int64,
	totalCredit int64,
	currency Currency,
	entries []StatementEntry,
	createdAt time.Time,
	updatedAt time.Time,
) *Statement {
	return &Statement{
		statementId:    id,
		accountId:      accountId,
		companyId:      companyId,
		initiatorId:    initiatorId,
		periodFrom:     periodFrom,
		periodTo:       periodTo,
		openingBalance: openingBalance,
		closingBalance: closingBalance,
		totalDebit:     totalDebit,
		totalCredit:    totalCredit,
		currency:       currency,
		entries:        entries,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

func (s *Statement) StatementId() uuid.UUID    { return s.statementId }
func (s *Statement) AccountId() uuid.UUID      { return s.accountId }
func (s *Statement) CompanyId() uuid.UUID      { return s.companyId }
func (s *Statement) InitiatorId() uuid.UUID    { return s.initiatorId }
func (s *Statement) PeriodFrom() time.Time     { return s.periodFrom }
func (s *Statement) PeriodTo() time.Time       { return s.periodTo }
func (s *Statement) OpeningBalance() int64     { return s.openingBalance }
func (s *Statement) ClosingBalance() int64     { return s.closingBalance }
func (s *Statement) TotalDebit() int64         { return s.totalDebit }
func (s *Statement) TotalCredit() int64        { return s.totalCredit }
func (s *Statement) Currency() Currency        { return s.currency }
func (s *Statement) Entries() []StatementEntry { return s.entries }
func (s *Statement) CreatedAt() time.Time      { return s.createdAt }
func (s *Statement) UpdatedAt() time.Time      { return s.updatedAt }

func (s *Statement) ToOutboxEvent() (*outbox.Event, error) {
	entriesJSON, err := json.Marshal(s.entries)
	if err != nil {
		return nil, fmt.Errorf("marshal statement entries: %w", err)
	}

	payload, err := json.Marshal(struct {
		StatementID    uuid.UUID       `json:"statement_id"`
		AccountID      uuid.UUID       `json:"account_id"`
		CompanyID      uuid.UUID       `json:"company_id"`
		InitiatorID    uuid.UUID       `json:"initiator_id"`
		PeriodFrom     time.Time       `json:"period_from"`
		PeriodTo       time.Time       `json:"period_to"`
		OpeningBalance int64           `json:"opening_balance"`
		ClosingBalance int64           `json:"closing_balance"`
		TotalDebit     int64           `json:"total_debit"`
		TotalCredit    int64           `json:"total_credit"`
		Currency       string          `json:"currency"`
		Entries        json.RawMessage `json:"entries"`
		CreatedAt      time.Time       `json:"created_at"`
	}{
		StatementID:    s.statementId,
		AccountID:      s.accountId,
		CompanyID:      s.companyId,
		InitiatorID:    s.initiatorId,
		PeriodFrom:     s.periodFrom,
		PeriodTo:       s.periodTo,
		OpeningBalance: s.openingBalance,
		ClosingBalance: s.closingBalance,
		TotalDebit:     s.totalDebit,
		TotalCredit:    s.totalCredit,
		Currency:       s.currency.String(),
		Entries:        entriesJSON,
		CreatedAt:      s.createdAt,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal statement: %w", err)
	}

	return outbox.WithType(&outbox.Event{
		AggregateID:   s.accountId,
		AggregateType: "statement",
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
	}, outbox.EventStatementGenerated), nil
}
