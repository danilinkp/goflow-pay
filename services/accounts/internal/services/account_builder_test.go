package services_test

import (
	"accounts/internal/domain/entities"
	"accounts/internal/services"

	"github.com/google/uuid"
)

type AccountBuilder struct {
	companyID uuid.UUID
	balance   int64
	currency  entities.Currency
	status    entities.AccountStatus
}

func NewAccountBuilder() *AccountBuilder {
	return &AccountBuilder{
		companyID: uuid.New(),
		balance:   10000,
		currency:  entities.RUB,
		status:    entities.ActiveStatus,
	}
}

func (b *AccountBuilder) WithCompanyID(id uuid.UUID) *AccountBuilder {
	b.companyID = id
	return b
}

func (b *AccountBuilder) WithBalance(balance int64) *AccountBuilder {
	b.balance = balance
	return b
}

func (b *AccountBuilder) WithCurrency(currency entities.Currency) *AccountBuilder {
	b.currency = currency
	return b
}

func (b *AccountBuilder) WithStatus(status entities.AccountStatus) *AccountBuilder {
	b.status = status
	return b
}

func (b *AccountBuilder) Build() *entities.Account {
	acc, err := entities.NewAccount(b.companyID, b.balance, b.currency, b.status)
	if err != nil {
		panic("AccountBuilder.Build: " + err.Error())
	}
	return acc
}

type BankOperationInputBuilder struct {
	accountID      uuid.UUID
	companyID      uuid.UUID
	bankAccountID  uuid.UUID
	initiatorID    uuid.UUID
	amount         int64
	idempotencyKey string
}

func NewBankOperationInputBuilder() *BankOperationInputBuilder {
	return &BankOperationInputBuilder{
		accountID:      uuid.New(),
		companyID:      uuid.New(),
		bankAccountID:  uuid.New(),
		initiatorID:    uuid.New(),
		amount:         5000,
		idempotencyKey: uuid.New().String(),
	}
}

func (b *BankOperationInputBuilder) WithAccountID(id uuid.UUID) *BankOperationInputBuilder {
	b.accountID = id
	return b
}

func (b *BankOperationInputBuilder) WithCompanyID(id uuid.UUID) *BankOperationInputBuilder {
	b.companyID = id
	return b
}

func (b *BankOperationInputBuilder) WithBankAccountID(id uuid.UUID) *BankOperationInputBuilder {
	b.bankAccountID = id
	return b
}

func (b *BankOperationInputBuilder) WithInitiatorID(id uuid.UUID) *BankOperationInputBuilder {
	b.initiatorID = id
	return b
}

func (b *BankOperationInputBuilder) WithAmount(amount int64) *BankOperationInputBuilder {
	b.amount = amount
	return b
}

func (b *BankOperationInputBuilder) WithIdempotencyKey(key string) *BankOperationInputBuilder {
	b.idempotencyKey = key
	return b
}

func (b *BankOperationInputBuilder) Build() *services.BankOperationInput {
	return &services.BankOperationInput{
		AccountID:      b.accountID,
		CompanyID:      b.companyID,
		BankAccountID:  b.bankAccountID,
		InitiatorID:    b.initiatorID,
		Amount:         b.amount,
		IdempotencyKey: b.idempotencyKey,
	}
}

type LinkBankInputBuilder struct {
	accountID         uuid.UUID
	companyID         uuid.UUID
	name              string
	bic               string
	settlementAccount string
	currency          entities.Currency
}

func NewLinkBankInputBuilder() *LinkBankInputBuilder {
	return &LinkBankInputBuilder{
		accountID:         uuid.New(),
		companyID:         uuid.New(),
		name:              "Сбербанк",
		bic:               "044525225",
		settlementAccount: "40702810200000000001",
		currency:          entities.RUB,
	}
}

func (b *LinkBankInputBuilder) WithAccountID(id uuid.UUID) *LinkBankInputBuilder {
	b.accountID = id
	return b
}

func (b *LinkBankInputBuilder) WithCompanyID(id uuid.UUID) *LinkBankInputBuilder {
	b.companyID = id
	return b
}

func (b *LinkBankInputBuilder) WithCurrency(currency entities.Currency) *LinkBankInputBuilder {
	b.currency = currency
	return b
}

func (b *LinkBankInputBuilder) WithBic(bic string) *LinkBankInputBuilder {
	b.bic = bic
	return b
}

func (b *LinkBankInputBuilder) Build() services.LinkBankInput {
	return services.LinkBankInput{
		AccountID:         b.accountID,
		CompanyID:         b.companyID,
		Name:              b.name,
		Bic:               b.bic,
		SettlementAccount: b.settlementAccount,
		Currency:          b.currency,
	}
}
