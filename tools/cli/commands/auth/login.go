package auth

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Token     string `json:"token"`
}

var emailFlag string

func NewLoginCmd(httpClient *client.Client) *cobra.Command {
	loginCmd := &cobra.Command{
		Use:   "login --email <email>",
		Short: "Login to GoFlow Pay",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print("Password: ")
			passwordBytes, err := term.ReadPassword(syscall.Stdin)
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}

			var resp loginResponse
			if err = httpClient.Post(context.Background(), "/api/v1/auth/login", loginRequest{
				Email:    emailFlag,
				Password: string(passwordBytes),
			}, &resp); err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			if err = credentials.Save(&credentials.Credentials{
				Token:     resp.Token,
				UserID:    resp.UserID,
				CompanyID: resp.CompanyID,
				Role:      resp.Role,
				Email:     resp.Email,
			}); err != nil {
				return fmt.Errorf("failed to save credentials: %w", err)
			}

			fmt.Printf("Logged in as %s (role: %s)\n", resp.Email, resp.Role)
			return nil
		},
	}

	loginCmd.Flags().StringVar(&emailFlag, "email", "", "Login to GoFlow Pay")
	_ = loginCmd.MarkFlagRequired("email")

	return loginCmd
}
