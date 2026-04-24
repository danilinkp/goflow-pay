package transactions

import (
	"cli/internal/client"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewGetTransactionCmd(httpClient *client.Client) *cobra.Command {
	var txIDFlag string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a transaction by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp transactionResponse
			if err := httpClient.Get(
				context.Background(),
				"/api/v1/transactions/detail/"+txIDFlag,
				&resp,
			); err != nil {
				return fmt.Errorf("failed to get transaction: %w", err)
			}

			fmt.Printf("Transaction:\n")
			fmt.Printf("  ID:       %s\n", resp.TransactionId)
			fmt.Printf("  Status:   %s\n", resp.TransactionStatus)
			fmt.Printf("  Amount:   %d %s\n", resp.Amount, resp.Currency)
			fmt.Printf("  From:     %s\n", resp.FromAccountId)
			fmt.Printf("  To:       %s\n", resp.ToAccountId)
			fmt.Printf("  Date:     %s\n", resp.CreatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}

	cmd.Flags().StringVar(&txIDFlag, "tx-id", "", "Transaction ID")
	_ = cmd.MarkFlagRequired("tx-id")
	return cmd
}
