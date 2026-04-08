package entities

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var (
	bicRe               = regexp.MustCompile(`^\d{9}$`)
	settlementAccountRe = regexp.MustCompile(`^\d{20}$`)

	weights = []int{7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1}
)

func ValidateBIC(bic string) bool {
	return bicRe.MatchString(bic)
}

func ValidateSettlementAccount(account, bic string) bool {
	if !settlementAccountRe.MatchString(account) {
		return false
	}
	if !ValidateBIC(bic) {
		return false
	}

	key := bic[6:] + account

	sum := 0
	for i, ch := range key {
		digit := int(ch - '0')
		sum += (digit * weights[i]) % 10
	}

	return sum%10 == 0
}

func ValidateSettlementAccountCurrency(account string, currency Currency) bool {
	return account[5:8] == currency.OKVCode()
}

type BankAccount struct {
	bankAccountId     uuid.UUID
	companyId         uuid.UUID
	name              string
	bic               string
	settlementAccount string
	currency          Currency
	createdAt         time.Time
	updatedAt         time.Time
}

func NewBankAccount(companyId uuid.UUID, name string, bic string, settlementAccount string, currency Currency) (*BankAccount, error) {
	if companyId == uuid.Nil {
		return nil, fmt.Errorf("%s: companyId is nil", "create bank account")
	}
	if name == "" {
		return nil, fmt.Errorf("%s: name is required", "create bank account")
	}
	if !ValidateBIC(bic) {
		return nil, fmt.Errorf("%s: invalid bic format", "create bank account")
	}
	if !settlementAccountRe.MatchString(settlementAccount) {
		return nil, fmt.Errorf("%s: settlement account must be 20 digits", "create bank account")
	}
	if !ValidateSettlementAccount(settlementAccount, bic) {
		return nil, fmt.Errorf("%s: invalid settlement account checksum", "create bank account")
	}
	if !currency.IsValid() {
		return nil, fmt.Errorf("%s: currency must be valid", "create bank account")
	}
	if !ValidateSettlementAccountCurrency(settlementAccount, currency) {
		return nil, fmt.Errorf("%s: settlement account currency does not match", "create bank account")
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

func ReconstructBankAccount(bankAccountId, companyId uuid.UUID, name, bic, settlementAccount string, currency Currency, createdAt, updatedAt time.Time) *BankAccount {
	return &BankAccount{
		bankAccountId:     bankAccountId,
		companyId:         companyId,
		name:              name,
		bic:               bic,
		settlementAccount: settlementAccount,
		currency:          currency,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}
}

func (ba *BankAccount) BankAccountId() uuid.UUID  { return ba.bankAccountId }
func (ba *BankAccount) CompanyId() uuid.UUID      { return ba.companyId }
func (ba *BankAccount) Name() string              { return ba.name }
func (ba *BankAccount) BIC() string               { return ba.bic }
func (ba *BankAccount) SettlementAccount() string { return ba.settlementAccount }
func (ba *BankAccount) Currency() Currency        { return ba.currency }
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

func (ba *BankAccount) UpdateCurrency(newCurrency Currency) error {
	if !ValidateSettlementAccountCurrency(ba.settlementAccount, newCurrency) {
		return fmt.Errorf("%s: settlement account currency does not match", "update bank account currency")
	}
	if !newCurrency.IsValid() {
		return fmt.Errorf("%s: new currency must be valid", "update bank account")
	}
	ba.currency = newCurrency
	ba.updatedAt = time.Now().UTC()
	return nil
}
