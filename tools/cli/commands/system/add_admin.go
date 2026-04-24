package system

import (
	"cli/internal/client"
	"cli/internal/infrastructure"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

type addAdminRequest struct {
	Login       string `json:"login"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	CompanyName string `json:"company_name"`
}

type authResponse struct {
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Token     string `json:"token"`
}

func NewAddAdminCmd(httpClient *client.Client) *cobra.Command {
	var login, email string

	cmd := &cobra.Command{
		Use:   "add-admin --login <login> -- <password>",
		Short: `Add an admin to a system (only admin)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print("Password for new admin: ")
			password, err := infrastructure.ReadPassword()
			if err != nil {
				return err
			}

			fmt.Print("Confirm password: ")
			confirm, err := infrastructure.ReadPassword()
			if err != nil {
				return err
			}

			if password != confirm {
				return fmt.Errorf("passwords do not match")
			}

			var resp authResponse
			if err = httpClient.Post(context.Background(), "/api/v1/system/admins",
				addAdminRequest{
					Login:    login,
					Email:    email,
					Password: password,
				}, &resp); err != nil {
				return fmt.Errorf("failed to add admin: %w", err)
			}

			fmt.Printf("Admin %s created successfully\n", resp.Email)
			return nil
		},
	}

	cmd.Flags().StringVar(&login, "login", "", "Admin login")
	cmd.Flags().StringVar(&email, "email", "", "Admin email")
	_ = cmd.MarkFlagRequired("login")
	_ = cmd.MarkFlagRequired("email")

	return cmd
}
