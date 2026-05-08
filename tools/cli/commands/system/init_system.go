package system

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"cli/internal/infrastructure"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

type systemInitRequest struct {
	BootstrapToken string `json:"bootstrap_token"`
	Login          string `json:"login"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	CompanyName    string `json:"company_name"`
}

type systemInitResponse struct {
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Token     string `json:"token"`
}

func NewSystemInitCmd(httpClient *client.Client) *cobra.Command {
	var bootstrapToken string
	var login, email string

	cmd := &cobra.Command{
		Use:   "init --bootstrap-token <bootstrap token> --login <login> --email <email> --companyName <company name>",
		Short: "Initialize the system with the first admin (one-time operation)",
		RunE: func(cmd *cobra.Command, args []string) error {
			password, err := infrastructure.ReadPassword()
			if err != nil {
				return err
			}

			var resp systemInitResponse
			if err = httpClient.Post(context.Background(), "/api/v1/system/init",
				systemInitRequest{
					BootstrapToken: bootstrapToken,
					Login:          login,
					Email:          email,
					Password:       password,
				}, &resp); err != nil {
				return fmt.Errorf("system init failed: %w", err)
			}

			credentials.Save(&credentials.Credentials{
				Token:     resp.Token,
				UserID:    resp.UserID,
				CompanyID: resp.CompanyID,
				Role:      resp.Role,
				Email:     resp.Email,
			})

			fmt.Println("System initialized successfully")
			fmt.Printf("Logged in as %s (role: %s)\n", resp.Email, resp.Role)
			return nil
		},
	}

	cmd.Flags().StringVar(&bootstrapToken, "bootstrap-token", "", "One-time bootstrap token")
	cmd.Flags().StringVar(&login, "login", "", "Admin login")
	cmd.Flags().StringVar(&email, "email", "", "Admin email")
	//cmd.Flags().StringVar(&companyName, "company-name", "", "Company name")
	_ = cmd.MarkFlagRequired("bootstrap-token")
	_ = cmd.MarkFlagRequired("login")
	_ = cmd.MarkFlagRequired("email")
	//_ = cmd.MarkFlagRequired("company-name")

	return cmd
}
