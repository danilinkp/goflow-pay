package companies

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

type userResponse struct {
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

func NewGetUsersCmd(httpClient *client.Client) *cobra.Command {
	var companyIDFlag string

	cmd := &cobra.Command{
		Use:   "users --company_id <company_id>",
		Short: `Get users by company id (only company admin or admin)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}

			url := "/api/v1/companies/users"
			if companyIDFlag != "" {
				if creds.Role != "admin" || creds.CompanyID != companyIDFlag {
					return fmt.Errorf("only admin can query other companies")
				}
				url += "?company_id=" + companyIDFlag
			}

			var resp []userResponse
			if err = httpClient.Get(
				context.Background(),
				url,
				&resp,
			); err != nil {
				return fmt.Errorf("failed to get users: %w", err)
			}

			for _, u := range resp {
				fmt.Printf("ID: %s | Login: %s | Email: %s | Role: %s\n",
					u.UserID, u.Login, u.Email, u.Role)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&companyIDFlag, "company-id", "", "ID of the company to query (System Admin only)")

	return cmd
}
