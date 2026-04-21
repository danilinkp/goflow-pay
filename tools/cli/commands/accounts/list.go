package accounts

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewGetAccountsCmd(httpClient *client.Client) *cobra.Command {
	var companyIDFlag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List accounts for your company",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}

			url := "/api/v1/companies/accounts"
			if companyIDFlag != "" {
				if creds.Role != "admin" {
					return fmt.Errorf("only admin can query other companies")
				}
				url += "?company_id=" + companyIDFlag
			}

			var resp []accountResponse
			if err = httpClient.Get(context.Background(), url, &resp); err != nil {
				return fmt.Errorf("failed to get accounts: %w", err)
			}

			if len(resp) == 0 {
				fmt.Println("No accounts found")
				return nil
			}

			for _, a := range resp {
				fmt.Printf("ID: %s | Balance: %d | Currency: %s | Status: %s\n",
					a.AccountId, a.Balance, a.Currency, a.Status)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&companyIDFlag, "company-id", "", "Company ID to query (admin only)")
	return cmd
}
