package companies

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewGetInviteCodeCmd(httpClient *client.Client) *cobra.Command {
	var companyIDFlag string

	cmd := &cobra.Command{
		Use:   "invite-code",
		Short: "Get invite code",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}

			url := "/api/v1/companies/invite-code"
			if companyIDFlag != "" {
				if creds.Role != "admin" || creds.CompanyID != companyIDFlag {
					return fmt.Errorf("only admin can get invite code of other companies")
				}
				url += "?company_id=" + companyIDFlag
			}

			finalID := companyIDFlag
			if finalID == "" {
				finalID = creds.CompanyID
			}

			if finalID == "" {
				return fmt.Errorf("company_id is required")
			}

			var inviteCode string
			if err = httpClient.Get(context.Background(), url, &inviteCode); err != nil {
				return fmt.Errorf("failed to get invite code: %w", err)
			}

			fmt.Printf("Invite Code for company %s:\n", finalID)
			fmt.Printf("Invite Code: %s\n", inviteCode)
			fmt.Println("\nShare this code with your teammates to register.")

			return nil
		},
	}

	cmd.Flags().StringVar(&companyIDFlag, "company-id", "", "Company ID (optional for company admins)")

	return cmd
}
