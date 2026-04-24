package accounts

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type bankOperationRequest struct {
	CompanyId      uuid.UUID `json:"companyId"`
	AccountId      uuid.UUID `json:"account_id"`
	BankAccountId  uuid.UUID `json:"bank_account_id"`
	Amount         int64     `json:"amount"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type bankOperationResponse struct {
	BankOperationId string `json:"bank_operation_id"`
	OperationType   string `json:"operation_type"`
	OperationStatus string `json:"operation_status"`
	Amount          int64  `json:"amount"`
	IdempotencyKey  string `json:"idempotency_key"`
}

func NewBankDepositCmd(httpClient *client.Client) *cobra.Command {
	var (
		accountID      string
		bankAccountID  string
		amount         int64
		idempotencyKey string
	)

	cmd := &cobra.Command{
		Use:   "bank-deposit",
		Short: "Make a bank deposit (company_admin only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}

			var resp bankOperationResponse
			if err = httpClient.Post(context.Background(), "/api/v1/accounts/bank/deposit",
				bankOperationRequest{
					CompanyId:      uuid.MustParse(creds.CompanyID),
					AccountId:      uuid.MustParse(accountID),
					BankAccountId:  uuid.MustParse(bankAccountID),
					Amount:         amount,
					IdempotencyKey: idempotencyKey,
				}, &resp,
			); err != nil {
				return fmt.Errorf("failed to make bank deposit: %w", err)
			}

			fmt.Printf("Bank deposit initiated:\n")
			fmt.Printf("  Operation ID: %s\n", resp.BankOperationId)
			fmt.Printf("  Status:       %s\n", resp.OperationStatus)
			fmt.Printf("  Amount:       %d\n", resp.Amount)
			return nil
		},
	}

	cmd.Flags().StringVar(&accountID, "account-id", "", "Account ID")
	cmd.Flags().StringVar(&bankAccountID, "bank-account-id", "", "Bank account ID")
	cmd.Flags().Int64Var(&amount, "amount", 0, "Amount to deposit")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	_ = cmd.MarkFlagRequired("account-id")
	_ = cmd.MarkFlagRequired("bank-account-id")
	_ = cmd.MarkFlagRequired("amount")
	_ = cmd.MarkFlagRequired("idempotency-key")

	return cmd
}
