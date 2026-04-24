package transactions

import (
	"cli/internal/client"

	"github.com/spf13/cobra"
)

func NewTransactionsCmd(httpClient *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transactions",
		Short: "Manage transactions",
	}

	cmd.AddCommand(NewTransferCmd(httpClient))
	cmd.AddCommand(NewGetAllTransactionsCmd(httpClient))
	cmd.AddCommand(NewGetTransactionCmd(httpClient))

	return cmd
}
