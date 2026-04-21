package companies

import (
	"cli/internal/client"

	"github.com/spf13/cobra"
)

func NewCompaniesCmd(httpClient *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "companies",
		Short: "Manage companies",
	}

	cmd.AddCommand(NewGetUsersCmd(httpClient))
	cmd.AddCommand(NewGetInviteCodeCmd(httpClient))
	cmd.AddCommand(NewGetAccountsCmd(httpClient))
	cmd.AddCommand(NewGetBanksCommand(httpClient))

	return cmd
}
