package accounts

import (
	"cli/internal/client"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type linkBankAccountRequest struct {
	AccountId         uuid.UUID `json:"account_id"`
	BankId            uuid.UUID `json:"bank_id"`
	Name              string    `json:"name"`
	BIC               string    `json:"bic"`
	SettlementAccount uuid.UUID `json:"settlement_account"`
	Currency          string    `json:"currency"`
}

type bankAccountResponse struct {
	BankAccountId     uuid.UUID `json:"bank_account_id"`
	CompanyId         uuid.UUID `json:"company_id"`
	Name              string    `json:"name"`
	BIC               string    `json:"bic"`
	SettlementAccount uuid.UUID `json:"settlement_account"`
	Currency          string    `json:"currency"`
	CreatedAt         time.Time `json:"created_at"`
}

func NewLinkBankAccountCmd(httpClient *client.Client) *cobra.Command {
	var (
		accountID         string
		bankID            string
		name              string
		bic               string
		settlementAccount string
		currency          string
	)

	cmd := &cobra.Command{
		Use:   "bank-link",
		Short: "Link a bank account (company_admin only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp bankAccountResponse
			if err := httpClient.Post(context.Background(), "/api/v1/accounts/bank",
				linkBankAccountRequest{
					AccountId:         uuid.MustParse(accountID),
					BankId:            uuid.MustParse(bankID),
					Name:              name,
					BIC:               bic,
					SettlementAccount: uuid.MustParse(settlementAccount),
					Currency:          currency,
				}, &resp,
			); err != nil {
				return fmt.Errorf("failed to link bank account: %w", err)
			}

			fmt.Printf("Bank account linked:\n")
			fmt.Printf("  ID:                 %s\n", resp.BankAccountId)
			fmt.Printf("  Name:               %s\n", resp.Name)
			fmt.Printf("  BIC:                %s\n", resp.BIC)
			fmt.Printf("  Settlement Account: %s\n", resp.SettlementAccount)
			fmt.Printf("  Currency:           %s\n", resp.Currency)
			return nil
		},
	}

	cmd.Flags().StringVar(&accountID, "account-id", "", "Account ID")
	cmd.Flags().StringVar(&bankID, "bank-id", "", "Bank ID")
	cmd.Flags().StringVar(&name, "name", "", "Bank account name")
	cmd.Flags().StringVar(&bic, "bic", "", "BIC code")
	cmd.Flags().StringVar(&settlementAccount, "settlement-account", "", "Settlement account")
	cmd.Flags().StringVar(&currency, "currency", "", "Currency (EUR, USD, RUB, CNY)")
	_ = cmd.MarkFlagRequired("account-id")
	_ = cmd.MarkFlagRequired("bank-id")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("bic")
	_ = cmd.MarkFlagRequired("settlement-account")
	_ = cmd.MarkFlagRequired("currency")
	return cmd
}
