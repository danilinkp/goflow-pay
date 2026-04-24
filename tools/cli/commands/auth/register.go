package auth

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"cli/internal/infrastructure"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

type registerWithNewCompanyRequest struct {
	Login       string `json:"login"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	CompanyName string `json:"company_name"`
}

type registerWithExistingCompanyRequest struct {
	Login      string `json:"login"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	InviteCode string `json:"invite_code"`
}

var (
	registerLogin       string
	registerEmail       string
	registerCompanyName string
	registerInviteCode  string
)

func NewRegisterNewCompanyCmd(httpClient *client.Client) *cobra.Command {
	registerNewCompanyCmd := &cobra.Command{
		Use:   "register-company --login <login> --email <email> --company-name <name>",
		Short: "Register a new company and become its admin",
		RunE: func(cmd *cobra.Command, args []string) error {
			password, err := infrastructure.ReadPassword()
			if err != nil {
				return err
			}

			var resp loginResponse
			if err = httpClient.Post(context.Background(), "/api/v1/auth/register/company", registerWithNewCompanyRequest{
				Login:       registerLogin,
				Email:       registerEmail,
				Password:    password,
				CompanyName: registerCompanyName,
			}, &resp); err != nil {
				return fmt.Errorf("registration failed: %w", err)
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

			fmt.Printf("Registered successfully as %s\n", resp.Email)
			fmt.Printf("Company ID: %s\n", resp.CompanyID)
			fmt.Printf("Role: %s\n", resp.Role)
			return nil
		},
	}

	registerNewCompanyCmd.Flags().StringVar(&registerLogin, "login", "", "User login")
	registerNewCompanyCmd.Flags().StringVar(&registerEmail, "email", "", "User email")
	registerNewCompanyCmd.Flags().StringVar(&registerCompanyName, "company-name", "", "Company name")
	_ = registerNewCompanyCmd.MarkFlagRequired("login")
	_ = registerNewCompanyCmd.MarkFlagRequired("email")
	_ = registerNewCompanyCmd.MarkFlagRequired("company-name")

	return registerNewCompanyCmd
}

func NewRegisterExistingCompanyCmd(httpClient *client.Client) *cobra.Command {
	registerExistingCompanyCmd := &cobra.Command{
		Use:   "register --login <login> --email <email> --invite-code <code>",
		Short: "Register and join an existing company by invite code",
		RunE: func(cmd *cobra.Command, args []string) error {
			password, err := infrastructure.ReadPassword()
			if err != nil {
				return err
			}

			var resp loginResponse
			if err = httpClient.Post(context.Background(), "/api/v1/auth/register", registerWithExistingCompanyRequest{
				Login:      registerLogin,
				Email:      registerEmail,
				Password:   password,
				InviteCode: registerInviteCode,
			}, &resp); err != nil {
				return fmt.Errorf("registration failed: %w", err)
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

			fmt.Printf("Registered successfully as %s\n", resp.Email)
			fmt.Printf("Company ID: %s\n", resp.CompanyID)
			fmt.Printf("Role: %s\n", resp.Role)
			return nil
		},
	}

	registerExistingCompanyCmd.Flags().StringVar(&registerLogin, "login", "", "User login")
	registerExistingCompanyCmd.Flags().StringVar(&registerEmail, "email", "", "User email")
	registerExistingCompanyCmd.Flags().StringVar(&registerInviteCode, "invite-code", "", "Company invite code")
	_ = registerExistingCompanyCmd.MarkFlagRequired("login")
	_ = registerExistingCompanyCmd.MarkFlagRequired("email")
	_ = registerExistingCompanyCmd.MarkFlagRequired("invite-code")

	return registerExistingCompanyCmd
}
