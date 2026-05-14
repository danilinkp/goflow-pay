package entities_test

import (
	"accounts/internal/domain/entities"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatement_Success(t *testing.T) {
	accountId := uuid.New()
	companyId := uuid.New()
	initiatorId := uuid.New()
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	entries := []entities.StatementEntry{
		{
			Date:         from,
			EntryType:    entities.EntryTypeBankDeposit,
			Amount:       500,
			BalanceAfter: 1500,
			Counterparty: "SberBank",
		},
	}

	s, err := entities.NewStatement(
		accountId, companyId, initiatorId,
		from, to,
		1000, 1500, 500, 0, entities.USD,
		entries,
	)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, s.StatementId())
	assert.Equal(t, accountId, s.AccountId())
	assert.Equal(t, companyId, s.CompanyId())
	assert.Equal(t, initiatorId, s.InitiatorId())
	assert.Equal(t, from, s.PeriodFrom())
	assert.Equal(t, to, s.PeriodTo())
	assert.Equal(t, int64(1000), s.OpeningBalance())
	assert.Equal(t, int64(1500), s.ClosingBalance())
	assert.Equal(t, int64(500), s.TotalDebit())
	assert.Equal(t, int64(0), s.TotalCredit())
	assert.Len(t, s.Entries(), 1)
	assert.False(t, s.CreatedAt().IsZero())
	assert.False(t, s.UpdatedAt().IsZero())
}

func TestNewStatement_NilAccountId(t *testing.T) {
	_, err := entities.NewStatement(
		uuid.Nil, uuid.New(), uuid.New(),
		time.Now(), time.Now().Add(time.Hour),
		0, 0, 0, 0, entities.USD, nil,
	)
	assert.Error(t, err)
}

func TestNewStatement_NilCompanyId(t *testing.T) {
	_, err := entities.NewStatement(
		uuid.New(), uuid.Nil, uuid.New(),
		time.Now(), time.Now().Add(time.Hour),
		0, 0, 0, 0, entities.USD, nil,
	)
	assert.Error(t, err)
}

func TestNewStatement_NilInitiatorId(t *testing.T) {
	_, err := entities.NewStatement(
		uuid.New(), uuid.New(), uuid.Nil,
		time.Now(), time.Now().Add(time.Hour),
		0, 0, 0, 0, entities.USD, nil,
	)
	assert.Error(t, err)
}

func TestNewStatement_ZeroPeriodFrom(t *testing.T) {
	_, err := entities.NewStatement(
		uuid.New(), uuid.New(), uuid.New(),
		time.Time{}, time.Now().Add(time.Hour),
		0, 0, 0, 0, entities.USD, nil,
	)
	assert.Error(t, err)
}

func TestNewStatement_ZeroPeriodTo(t *testing.T) {
	_, err := entities.NewStatement(
		uuid.New(), uuid.New(), uuid.New(),
		time.Now(), time.Time{},
		0, 0, 0, 0, entities.USD, nil,
	)
	assert.Error(t, err)
}

func TestNewStatement_PeriodFromAfterPeriodTo(t *testing.T) {
	from := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := entities.NewStatement(
		uuid.New(), uuid.New(), uuid.New(),
		from, to,
		0, 0, 0, 0, entities.USD, nil,
	)
	assert.Error(t, err)
}

func TestNewStatement_PeriodFromEqualPeriodTo_Success(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := entities.NewStatement(
		uuid.New(), uuid.New(), uuid.New(),
		now, now,
		0, 0, 0, 0, entities.USD, nil,
	)
	assert.NoError(t, err)
}

func TestNewStatement_NilEntries_Success(t *testing.T) {
	s, err := newTestStatement(t, nil)

	require.NoError(t, err)
	assert.Empty(t, s.Entries())
}

func TestNewStatement_EmptyEntries_Success(t *testing.T) {
	s, err := newTestStatement(t, []entities.StatementEntry{})

	require.NoError(t, err)
	assert.Empty(t, s.Entries())
}

func TestNewStatement_MultipleEntries(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	entries := []entities.StatementEntry{
		{Date: from, EntryType: entities.EntryTypeBankDeposit, Amount: 500, BalanceAfter: 1500, Counterparty: "Bank A"},
		{Date: from.Add(24 * time.Hour), EntryType: entities.EntryTypeTransferOut, Amount: 200, BalanceAfter: 1300, Counterparty: "Company B"},
		{Date: from.Add(48 * time.Hour), EntryType: entities.EntryTypeTransferIn, Amount: 100, BalanceAfter: 1400, Counterparty: "Company C"},
	}

	s, err := newTestStatement(t, entries)

	require.NoError(t, err)
	assert.Len(t, s.Entries(), 3)
}

func TestStatement_ToOutboxEvent_Success(t *testing.T) {
	entries := []entities.StatementEntry{
		{
			Date:         time.Now().UTC(),
			EntryType:    entities.EntryTypeBankDeposit,
			Amount:       500,
			BalanceAfter: 1500,
			Counterparty: "SberBank",
		},
	}

	s, err := newTestStatement(t, entries)
	require.NoError(t, err)

	evt, err := s.ToOutboxEvent()

	require.NoError(t, err)
	require.NotNil(t, evt)
	assert.NotEmpty(t, evt.Payload)
	assert.Equal(t, s.AccountId(), evt.AggregateID)
	assert.Equal(t, "statement", evt.AggregateType)
}

func TestStatement_ToOutboxEvent_EmptyEntries(t *testing.T) {
	s, err := newTestStatement(t, nil)
	require.NoError(t, err)

	evt, err := s.ToOutboxEvent()

	require.NoError(t, err)
	require.NotNil(t, evt)
	assert.NotEmpty(t, evt.Payload)
}

func TestReconstructStatement_PreservesAllFields(t *testing.T) {
	id := uuid.New()
	accountId := uuid.New()
	companyId := uuid.New()
	initiatorId := uuid.New()
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 2, 2, 12, 0, 0, 0, time.UTC)
	entries := []entities.StatementEntry{
		{Date: from, EntryType: entities.EntryTypeTransferIn, Amount: 300, BalanceAfter: 1300, Counterparty: "X"},
	}

	s := entities.ReconstructStatement(
		id, accountId, companyId, initiatorId,
		from, to,
		1000, 1300, 300, 0, entities.USD,
		entries,
		createdAt, updatedAt,
	)

	assert.Equal(t, id, s.StatementId())
	assert.Equal(t, accountId, s.AccountId())
	assert.Equal(t, companyId, s.CompanyId())
	assert.Equal(t, initiatorId, s.InitiatorId())
	assert.Equal(t, from, s.PeriodFrom())
	assert.Equal(t, to, s.PeriodTo())
	assert.Equal(t, int64(1000), s.OpeningBalance())
	assert.Equal(t, int64(1300), s.ClosingBalance())
	assert.Equal(t, int64(300), s.TotalDebit())
	assert.Equal(t, int64(0), s.TotalCredit())
	assert.Equal(t, entries, s.Entries())
	assert.Equal(t, createdAt, s.CreatedAt())
	assert.Equal(t, updatedAt, s.UpdatedAt())
}

func newTestStatement(t *testing.T, entries []entities.StatementEntry) (*entities.Statement, error) {
	t.Helper()
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	return entities.NewStatement(
		uuid.New(), uuid.New(), uuid.New(),
		from, to,
		1000, 1500, 500, 0, entities.USD,
		entries,
	)
}
