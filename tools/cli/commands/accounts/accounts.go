package accounts

import (
	"cli/internal/client"

	"github.com/spf13/cobra"
)

func NewAccountsCmd(httpClient *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Manage accounts",
	}

	cmd.AddCommand(NewCreateAccountCmd(httpClient))
	cmd.AddCommand(NewGetAccountsCmd(httpClient))
	cmd.AddCommand(NewGetBalanceCmd(httpClient))
	cmd.AddCommand(NewDeactivateAccountCmd(httpClient))
	cmd.AddCommand(NewLinkBankAccountCmd(httpClient))
	cmd.AddCommand(NewGetBankAccountsCmd(httpClient))
	cmd.AddCommand(NewBankDepositCmd(httpClient))
	cmd.AddCommand(NewBankWithdrawalCmd(httpClient))

	return cmd
}
