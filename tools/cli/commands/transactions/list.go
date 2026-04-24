package transactions

import (
	"cli/internal/client"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewGetAllTransactionsCmd(httpClient *client.Client) *cobra.Command {
	var accountIDFlag string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all transactions for an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp []transactionResponse
			if err := httpClient.Get(
				context.Background(),
				"/api/v1/transactions/"+accountIDFlag,
				&resp,
			); err != nil {
				return fmt.Errorf("failed to get transactions: %w", err)
			}

			if len(resp) == 0 {
				fmt.Println("No transactions found")
				return nil
			}

			for _, t := range resp {
				fmt.Printf("ID: %s | Amount: %d %s | Status: %s | Date: %s\n",
					t.TransactionId,
					t.Amount,
					t.Currency,
					t.TransactionStatus,
					t.CreatedAt.Format("2006-01-02 15:04:05"),
				)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&accountIDFlag, "account-id", "", "Account ID")
	_ = cmd.MarkFlagRequired("account-id")
	return cmd
}
