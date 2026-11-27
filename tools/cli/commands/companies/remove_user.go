package companies

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewRemoveUserCmd(httpClient *client.Client) *cobra.Command {
	var (
		companyIDFlag string
		userIDFlag    string
	)

	cmd := &cobra.Command{
		Use:   "remove-user --user-id <user_id>",
		Short: "Remove (fire) a user from company (company_admin or admin only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}

			targetCompanyID := creds.CompanyID
			if companyIDFlag != "" {
				if creds.Role != "admin" && creds.CompanyID != companyIDFlag {
					return fmt.Errorf("only admin can remove users from other companies")
				}
				targetCompanyID = companyIDFlag
			}

			if targetCompanyID == "" {
				return fmt.Errorf("company-id is required")
			}

			url := fmt.Sprintf("/api/v2/companies/%s/users/%s", targetCompanyID, userIDFlag)
			if err = httpClient.Delete(context.Background(), url, nil); err != nil {
				return fmt.Errorf("failed to remove user: %w", err)
			}

			fmt.Printf("User %s successfully removed from company %s\n", userIDFlag, targetCompanyID)
			return nil
		},
	}

	cmd.Flags().StringVar(&userIDFlag, "user-id", "", "User ID to remove")
	cmd.Flags().StringVar(&companyIDFlag, "company-id", "", "Company ID (optional for company_admin)")
	_ = cmd.MarkFlagRequired("user-id")

	return cmd
}
