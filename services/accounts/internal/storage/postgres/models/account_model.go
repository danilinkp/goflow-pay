package models

import (
	"accounts/internal/domain/entities"
	"time"

	"github.com/google/uuid"
)

type AccountModel struct {
	AccountId uuid.UUID `db:"account_id"`
	CompanyId uuid.UUID `db:"company_id"`
	Balance   int64     `db:"balance"`
	Currency  string    `db:"currency"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (a *AccountModel) ToDomain() *entities.Account {
	return entities.ReconstructAccount(a.AccountId, a.CompanyId, a.Balance, entities.Currency(a.Currency), entities.AccountStatus(a.Status), a.CreatedAt, a.UpdatedAt)
}

func ToAccountModel(en *entities.Account) *AccountModel {
	return &AccountModel{
		AccountId: en.AccountId(),
		CompanyId: en.CompanyId(),
		Balance:   en.Balance(),
		Currency:  en.Currency().String(),
		Status:    en.Status().String(),
		CreatedAt: en.CreatedAt(),
		UpdatedAt: en.UpdatedAt(),
	}
}
