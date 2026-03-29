package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type BankAccount struct {
	bankAccountId     uuid.UUID
	companyId         uuid.UUID
	name              string
	bic               string
	settlementAccount string
	currency          string
	createdAt         time.Time
	updatedAt         time.Time
}

func NewBankAccount(companyId uuid.UUID, name string, bic string, settlementAccount string, currency string) (*BankAccount, error) {
	if companyId == uuid.Nil {
		return nil, fmt.Errorf("%s: companyId is nil", "create bank account")
	}
	if name == "" {
		return nil, fmt.Errorf("%s: name is required", "create bank account")
	}
	if bic == "" {
		return nil, fmt.Errorf("%s: bic is required", "create bank account")
	}
	if settlementAccount == "" {
		return nil, fmt.Errorf("%s: settlementAccount is required", "create bank account")
	}
	if currency == "" {
		return nil, fmt.Errorf("%s: currency is required", "create bank account")
	}

	now := time.Now().UTC()
	return &BankAccount{
		bankAccountId:     uuid.New(),
		companyId:         companyId,
		name:              name,
		bic:               bic,
		settlementAccount: settlementAccount,
		currency:          currency,
		createdAt:         now,
		updatedAt:         now,
	}, nil
}

func (ba *BankAccount) BankAccountId() uuid.UUID  { return ba.bankAccountId }
func (ba *BankAccount) CompanyId() uuid.UUID      { return ba.companyId }
func (ba *BankAccount) Name() string              { return ba.name }
func (ba *BankAccount) BIC() string               { return ba.bic }
func (ba *BankAccount) SettlementAccount() string { return ba.settlementAccount }
func (ba *BankAccount) Currency() string          { return ba.currency }
func (ba *BankAccount) CreatedAt() time.Time      { return ba.createdAt }
func (ba *BankAccount) UpdatedAt() time.Time      { return ba.updatedAt }

func (ba *BankAccount) UpdateName(newName string) error {
	if newName == "" {
		return fmt.Errorf("%s: new name is required", "update bank account")
	}
	ba.name = newName
	ba.updatedAt = time.Now().UTC()
	return nil
}

func (ba *BankAccount) UpdateBIC(newBIC string) error {
	if newBIC == "" {
		return fmt.Errorf("%s: new bic is required", "update bank account")
	}
	ba.bic = newBIC
	ba.updatedAt = time.Now().UTC()
	return nil
}

func (ba *BankAccount) UpdateSettlementAccount(newSettlementAccount string) error {
	if newSettlementAccount == "" {
		return fmt.Errorf("%s: new bic is required", "update bank account")
	}
	ba.settlementAccount = newSettlementAccount
	ba.updatedAt = time.Now().UTC()
	return nil
}

func (ba *BankAccount) UpdateCurrency(newCurrency string) error {
	if newCurrency == "" {
		return fmt.Errorf("%s: new bic is required", "update bank account")
	}
	ba.currency = newCurrency
	ba.updatedAt = time.Now().UTC()
	return nil
}
