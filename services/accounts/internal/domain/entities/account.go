package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AccountStatus string

const (
	ActiveStatus   AccountStatus = "active"
	InactiveStatus AccountStatus = "inactive"
)

func (as AccountStatus) String() string {
	return string(as)
}

func (as AccountStatus) IsValid() bool {
	return as == ActiveStatus || as == InactiveStatus
}

type Account struct {
	accountId uuid.UUID
	companyId uuid.UUID
	balance   int64
	currency  Currency
	status    AccountStatus
	createdAt time.Time
	updatedAt time.Time
}

func NewAccount(companyId uuid.UUID, balance int64, currency Currency, status AccountStatus) (*Account, error) {
	if balance < 0 {
		return nil, fmt.Errorf("%s: account balance cannot be negative", "create account")
	}
	if !currency.IsValid() {
		return nil, fmt.Errorf("%s: account currency must be valid", "create account")
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("%s: account status must be valid", "create account")
	}

	now := time.Now().UTC()
	return &Account{
		accountId: uuid.New(),
		companyId: companyId,
		balance:   balance,
		currency:  currency,
		status:    status,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func ReconstructAccount(accountId uuid.UUID, companyId uuid.UUID, balance int64, currency Currency, status AccountStatus, createdAt, updatedAt time.Time) *Account {
	return &Account{
		accountId: accountId,
		companyId: companyId,
		balance:   balance,
		currency:  currency,
		status:    status,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (a *Account) AccountId() uuid.UUID  { return a.accountId }
func (a *Account) CompanyId() uuid.UUID  { return a.companyId }
func (a *Account) Balance() int64        { return a.balance }
func (a *Account) Currency() Currency    { return a.currency }
func (a *Account) Status() AccountStatus { return a.status }
func (a *Account) CreatedAt() time.Time  { return a.createdAt }
func (a *Account) UpdatedAt() time.Time  { return a.updatedAt }

func (a *Account) UpdateBalance(newBalance int64) error {
	if newBalance < 0 {
		return fmt.Errorf("%s: account balance cannot be negative", "update account")
	}
	a.balance = newBalance
	a.updatedAt = time.Now().UTC()
	return nil
}

func (a *Account) AddBalance(amount int64) error {
	if amount < 0 {
		return fmt.Errorf("%s: account balance cannot be negative", "add account")
	}
	a.balance += amount
	a.updatedAt = time.Now().UTC()
	return nil
}

func (a *Account) RemoveBalance(amount int64) error {
	if amount < 0 {
		return fmt.Errorf("%s: account balance cannot be negative", "remove account")
	}
	a.balance -= amount
	a.updatedAt = time.Now().UTC()
	return nil
}

func (a *Account) UpdateCurrency(newCurrency Currency) error {
	if !newCurrency.IsValid() {
		return fmt.Errorf("%s: account currency must be valid", "update account")
	}
	a.currency = newCurrency
	a.updatedAt = time.Now().UTC()
	return nil
}

func (a *Account) UpdateStatus(newStatus AccountStatus) error {
	if !newStatus.IsValid() {
		return fmt.Errorf("%s: account status must be valid", "update account")
	}
	a.status = newStatus
	a.updatedAt = time.Now().UTC()
	return nil
}
