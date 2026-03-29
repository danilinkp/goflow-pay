package request

import "github.com/google/uuid"

type LinkBankRequest struct {
	AccountID         uuid.UUID
	BankID            uuid.UUID
	CompanyID         uuid.UUID
	Name              string
	Bic               string
	SettlementAccount string
	Currency          string
}
