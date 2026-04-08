package models

import (
	"accounts/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type BankAccountModel struct {
	BankAccountId     uuid.UUID `db:"bank_account_id"`
	CompanyId         uuid.UUID `db:"company_id"`
	Name              string    `db:"name"`
	Bic               string    `db:"bic"`
	SettlementAccount string    `db:"settlement_account"`
	Currency          string    `db:"currency"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

func (m *BankAccountModel) ToDomain() *entities.BankAccount {
	return entities.ReconstructBankAccount(m.BankAccountId, m.CompanyId, m.Name, m.Bic, m.SettlementAccount, entities.Currency(m.Currency), m.CreatedAt, m.UpdatedAt)
}

func ToBankAccountModel(from *entities.BankAccount) *BankAccountModel {
	return &BankAccountModel{
		BankAccountId:     from.BankAccountId(),
		CompanyId:         from.CompanyId(),
		Name:              from.Name(),
		Bic:               from.BIC(),
		SettlementAccount: from.SettlementAccount(),
		Currency:          from.Currency().String(),
		CreatedAt:         from.CreatedAt(),
		UpdatedAt:         from.UpdatedAt(),
	}
}
