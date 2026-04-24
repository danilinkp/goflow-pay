package accounts

import (
	"cli/internal/client"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

type balanceResponse struct {
	Balance int64 `json:"balance"`
}

func NewGetBalanceCmd(httpClient *client.Client) *cobra.Command {
	var accountIDFlag string

	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Get balance for an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp balanceResponse
			if err := httpClient.Get(
				context.Background(),
				"/api/v1/accounts/"+accountIDFlag+"/balance",
				&resp,
			); err != nil {
				return fmt.Errorf("failed to get balance: %w", err)
			}

			fmt.Printf("Balance: %d\n", resp.Balance)
			return nil
		},
	}

	cmd.Flags().StringVar(&accountIDFlag, "account-id", "", "Account ID")
	_ = cmd.MarkFlagRequired("account-id")
	return cmd
}
