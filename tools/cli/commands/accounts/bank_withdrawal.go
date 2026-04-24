package accounts

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func NewBankWithdrawalCmd(httpClient *client.Client) *cobra.Command {
	var (
		accountID      string
		bankAccountID  string
		amount         int64
		idempotencyKey string
	)

	cmd := &cobra.Command{
		Use:   "bank-withdrawal",
		Short: "Make a bank withdrawal (company_admin or admin only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}

			var resp bankOperationResponse
			if err = httpClient.Post(context.Background(), "/api/v1/accounts/bank/withdrawal",
				bankOperationRequest{
					CompanyId:      uuid.MustParse(creds.CompanyID),
					AccountId:      uuid.MustParse(accountID),
					BankAccountId:  uuid.MustParse(bankAccountID),
					Amount:         amount,
					IdempotencyKey: idempotencyKey,
				}, &resp,
			); err != nil {
				return fmt.Errorf("failed to make bank withdrawal: %w", err)
			}

			fmt.Printf("Bank withdrawal initiated:\n")
			fmt.Printf("  Operation ID: %s\n", resp.BankOperationId)
			fmt.Printf("  Status:       %s\n", resp.OperationStatus)
			fmt.Printf("  Amount:       %d\n", resp.Amount)
			return nil
		},
	}

	cmd.Flags().StringVar(&accountID, "account-id", "", "Account ID")
	cmd.Flags().StringVar(&bankAccountID, "bank-account-id", "", "Bank account ID")
	cmd.Flags().Int64Var(&amount, "amount", 0, "Amount to withdraw")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	_ = cmd.MarkFlagRequired("account-id")
	_ = cmd.MarkFlagRequired("bank-account-id")
	_ = cmd.MarkFlagRequired("amount")
	_ = cmd.MarkFlagRequired("idempotency-key")

	return cmd
}
