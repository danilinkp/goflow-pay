package models

import (
	"accounts/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type BankAccountModel struct {
	BankAccountId     uuid.UUID `bson:"_id"`
	CompanyId         uuid.UUID `bson:"company_id"`
	Name              string    `bson:"name"`
	Bic               string    `bson:"bic"`
	SettlementAccount string    `bson:"settlement_account"`
	Currency          string    `bson:"currency"`
	CreatedAt         time.Time `bson:"created_at"`
	UpdatedAt         time.Time `bson:"updated_at"`
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
