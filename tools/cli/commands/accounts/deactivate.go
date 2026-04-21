package accounts

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewDeactivateAccountCmd(httpClient *client.Client) *cobra.Command {
	var accountIDFlag, companyIDFlag string

	cmd := &cobra.Command{
		Use:   "deactivate",
		Short: "Deactivate an account (company_admin or admin only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}

			url := "/api/v1/accounts/" + accountIDFlag
			if companyIDFlag != "" {
				if creds.Role != "admin" {
					return fmt.Errorf("only admin can deactivate accounts of other companies")
				}
				url += "?company_id=" + companyIDFlag
			}

			var resp accountResponse
			if err = httpClient.Delete(
				context.Background(),
				url,
				&resp,
			); err != nil {
				return fmt.Errorf("failed to deactivate account: %w", err)
			}

			fmt.Printf("Account %s deactivated\n", resp.AccountId)
			fmt.Printf("  Status: %s\n", resp.Status)
			return nil
		},
	}

	cmd.Flags().StringVar(&accountIDFlag, "account-id", "", "Account ID to deactivate")
	cmd.Flags().StringVar(&companyIDFlag, "company-id", "", "Company ID (only for system admins)")
	_ = cmd.MarkFlagRequired("account-id")
	return cmd
}
