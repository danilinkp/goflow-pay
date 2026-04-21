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

type accountResponse struct {
	AccountId uuid.UUID `json:"account_id"`
	CompanyId uuid.UUID `json:"company_id"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func NewGetAccountsCmd(httpClient *client.Client) *cobra.Command {
	var companyIDFlag string

	cmd := &cobra.Command{
		Use:   "get-accounts --company_id <company_id>",
		Short: "Get accounts for a company",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}

			url := "/api/v1/companies/accounts"
			if companyIDFlag != "" {
				if creds.Role != "admin" || creds.CompanyID != companyIDFlag {
					return fmt.Errorf("only admin can get accounts of other companies")
				}
				url += "?company_id=" + companyIDFlag
			}

			var resp []accountResponse
			err = httpClient.Get(context.Background(), url, &resp)
			if err != nil {
				return fmt.Errorf("cannot get accounts: %w", err)
			}

			for _, a := range resp {
				fmt.Printf("ID:        %s\n", a.AccountId)
				fmt.Printf("Balance:   %d %s\n", a.Balance, a.Currency)
				fmt.Printf("Status:    %s\n", a.Status)
				fmt.Printf("Created:   %s\n", a.CreatedAt.Format("2006-01-02 15:04:05"))
				fmt.Println("---")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&companyIDFlag, "company_id", "", "company ID (System Admin Only")
	return cmd
}
