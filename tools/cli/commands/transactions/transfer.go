package transactions

import (
	"cli/internal/client"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type transferRequest struct {
	FromAccountId  uuid.UUID `json:"from_account_id"`
	ToAccountId    uuid.UUID `json:"to_account_id"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type transactionResponse struct {
	TransactionId     uuid.UUID `json:"transaction_id"`
	InitiatorId       uuid.UUID `json:"initiator_id"`
	FromAccountId     uuid.UUID `json:"from_account_id"`
	ToAccountId       uuid.UUID `json:"to_account_id"`
	Amount            int64     `json:"amount"`
	Currency          string    `json:"currency"`
	IdempotencyKey    string    `json:"idempotency_key"`
	TransactionStatus string    `json:"transaction_status"`
	CreatedAt         time.Time `json:"created_at"`
}

func NewTransferCmd(httpClient *client.Client) *cobra.Command {
	var (
		fromAccountID  string
		toAccountID    string
		amount         int64
		currency       string
		idempotencyKey string
	)

	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer funds between accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp transactionResponse
			if err := httpClient.Post(context.Background(), "/api/v1/transactions/transfer",
				transferRequest{
					FromAccountId:  uuid.MustParse(fromAccountID),
					ToAccountId:    uuid.MustParse(toAccountID),
					Amount:         amount,
					Currency:       currency,
					IdempotencyKey: idempotencyKey,
				}, &resp,
			); err != nil {
				return fmt.Errorf("transfer failed: %w", err)
			}

			fmt.Printf("Transfer initiated:\n")
			fmt.Printf("  Transaction ID: %s\n", resp.TransactionId)
			fmt.Printf("  Status:         %s\n", resp.TransactionStatus)
			fmt.Printf("  Amount:         %d %s\n", resp.Amount, resp.Currency)
			fmt.Printf("  From:           %s\n", resp.FromAccountId)
			fmt.Printf("  To:             %s\n", resp.ToAccountId)
			return nil
		},
	}

	cmd.Flags().StringVar(&fromAccountID, "from", "", "Source account ID")
	cmd.Flags().StringVar(&toAccountID, "to", "", "Destination account ID")
	cmd.Flags().Int64Var(&amount, "amount", 0, "Amount to transfer")
	cmd.Flags().StringVar(&currency, "currency", "", "Currency (EUR, USD, RUB, CNY)")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("amount")
	_ = cmd.MarkFlagRequired("currency")
	_ = cmd.MarkFlagRequired("idempotency-key")

	return cmd
}
