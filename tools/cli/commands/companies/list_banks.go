package companies

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type bankResponse struct {
	BankAccountId     uuid.UUID `json:"bank_account_id"`
	CompanyId         uuid.UUID `json:"company_id"`
	Name              string    `json:"name"`
	BIC               string    `json:"bic"`
	SettlementAccount uuid.UUID `json:"settlement_account"`
	Currency          string    `json:"currency"`
	CreatedAt         time.Time `json:"created_at"`
}

func NewGetBanksCommand(httpClient *client.Client) *cobra.Command {
	var companyIDFlag string

	cmd := &cobra.Command{
		Use:   "get-banks --company_id <company_id>",
		Short: "Get banks for a company",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}

			url := "/api/v1/companies/banks"
			if companyIDFlag != "" {
				if creds.Role != "admin" || creds.CompanyID != companyIDFlag {
					return fmt.Errorf("only admin can get bank accounts of other companies")
				}
				url += "?company_id=" + companyIDFlag
			}

			var resp []bankResponse
			err = httpClient.Get(context.Background(), url, &resp)
			if err != nil {
				return fmt.Errorf("cannot get accounts: %w", err)
			}

			for _, b := range resp {
				fmt.Printf("ID: %s\n", b.BankAccountId)
				fmt.Printf("Name: %s\n", b.Name)
				fmt.Printf("BIC: %s\n", b.BIC)
				fmt.Printf("Settlement account: %s\n", b.SettlementAccount)
				fmt.Printf("Currency: %s\n", b.Currency)
				fmt.Printf("CreatedAt: %s\n", b.CreatedAt.Format("2006-01-02 15:04:05"))
				fmt.Println("---")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&companyIDFlag, "company_id", "", "company ID (System Admin Only")
	return cmd
}
