package accounts

import (
	"cli/internal/client"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type accountResponse struct {
	AccountId uuid.UUID `json:"account_id"`
	CompanyId uuid.UUID `json:"company_id"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type createAccountRequest struct {
	Currency string `json:"currency"`
}

func NewCreateAccountCmd(httpClient *client.Client) *cobra.Command {
	var currency string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new account (company_admin only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp accountResponse
			if err := httpClient.Post(context.Background(), "/api/v1/accounts",
				createAccountRequest{Currency: currency},
				&resp,
			); err != nil {
				return fmt.Errorf("failed to create account: %w", err)
			}

			fmt.Printf("Account created:\n")
			fmt.Printf("  ID:       %s\n", resp.AccountId)
			fmt.Printf("  Currency: %s\n", resp.Currency)
			fmt.Printf("  Status:   %s\n", resp.Status)
			return nil
		},
	}

	cmd.Flags().StringVar(&currency, "currency", "", "Account currency (EUR, USD, RUB, CNY)")
	_ = cmd.MarkFlagRequired("currency")

	return cmd
}
