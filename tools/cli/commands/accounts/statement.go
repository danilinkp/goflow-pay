package accounts

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type statementResponse struct {
	StatementId    uuid.UUID `json:"statement_id"`
	AccountId      uuid.UUID `json:"account_id"`
	CompanyId      uuid.UUID `json:"company_id"`
	InitiatorId    uuid.UUID `json:"initiator_id"`
	PeriodFrom     time.Time `json:"period_from"`
	PeriodTo       time.Time `json:"period_to"`
	OpeningBalance int64     `json:"opening_balance"`
	ClosingBalance int64     `json:"closing_balance"`
	TotalDebit     int64     `json:"total_debit"`
	TotalCredit    int64     `json:"total_credit"`
	CreatedAt      time.Time `json:"created_at"`
}

type generateStatementRequest struct {
	PeriodFrom time.Time `json:"period_from"`
	PeriodTo   time.Time `json:"period_to"`
}

func NewGenerateStatementCmd(httpClient *client.Client) *cobra.Command {
	var accountIDFlag, fromFlag, toFlag string

	cmd := &cobra.Command{
		Use:   "statement",
		Short: "Generate account statement and send to email (all authenticated users)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := credentials.Load()
			if err != nil {
				return err
			}

			periodFrom, err := time.Parse("2006-01-02", fromFlag)
			if err != nil {
				return fmt.Errorf("invalid from date format, use YYYY-MM-DD: %w", err)
			}

			periodTo, err := time.Parse("2006-01-02", toFlag)
			if err != nil {
				return fmt.Errorf("invalid to date format, use YYYY-MM-DD: %w", err)
			}

			if periodFrom.After(periodTo) {
				return fmt.Errorf("from date must be before to date")
			}

			var resp statementResponse
			if err = httpClient.Post(
				context.Background(),
				"/api/v1/accounts/"+accountIDFlag+"/statement",
				generateStatementRequest{
					PeriodFrom: periodFrom,
					PeriodTo:   periodTo,
				},
				&resp,
			); err != nil {
				return fmt.Errorf("failed to generate statement: %w", err)
			}

			fmt.Printf("Statement generated successfully\n")
			fmt.Printf("  Statement ID:    %s\n", resp.StatementId)
			fmt.Printf("  Account ID:      %s\n", resp.AccountId)
			fmt.Printf("  Period:          %s — %s\n",
				resp.PeriodFrom.Format("02.01.2006"),
				resp.PeriodTo.Format("02.01.2006"),
			)
			fmt.Printf("  Opening Balance: %d\n", resp.OpeningBalance)
			fmt.Printf("  Closing Balance: %d\n", resp.ClosingBalance)
			fmt.Printf("  Total Debit:     %d\n", resp.TotalDebit)
			fmt.Printf("  Total Credit:    %d\n", resp.TotalCredit)
			fmt.Println("\nStatement has been sent to your email.")
			return nil
		},
	}

	cmd.Flags().StringVar(&accountIDFlag, "account-id", "", "Account ID")
	cmd.Flags().StringVar(&fromFlag, "from", "", "Period start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&toFlag, "to", "", "Period end date (YYYY-MM-DD)")
	_ = cmd.MarkFlagRequired("account-id")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}
