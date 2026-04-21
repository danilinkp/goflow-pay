package accounts

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewGetBankAccountsCmd(httpClient *client.Client) *cobra.Command {
	var companyIDFlag string

	cmd := &cobra.Command{
		Use:   "bank-list",
		Short: "List bank accounts for your company",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}

			url := "/api/v1/companies/banks"
			if companyIDFlag != "" {
				if creds.Role != "admin" {
					return fmt.Errorf("only admin can query other companies")
				}
				url += "?company_id=" + companyIDFlag
			}

			var resp []bankAccountResponse
			if err = httpClient.Get(context.Background(), url, &resp); err != nil {
				return fmt.Errorf("failed to get bank accounts: %w", err)
			}

			if len(resp) == 0 {
				fmt.Println("No bank accounts found")
				return nil
			}

			for _, ba := range resp {
				fmt.Printf("ID: %s | Name: %s | BIC: %s | Currency: %s\n",
					ba.BankAccountId, ba.Name, ba.BIC, ba.Currency)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&companyIDFlag, "company-id", "", "Company ID to query (admin only)")
	return cmd
}
