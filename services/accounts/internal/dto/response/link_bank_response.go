package response

import "github.com/google/uuid"

type LinkBankResponse struct {
	AccountID     uuid.UUID
	BankAccountID uuid.UUID
	BankName      string
}
