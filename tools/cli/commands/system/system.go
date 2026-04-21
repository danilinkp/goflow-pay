package system

import (
	"cli/internal/client"

	"github.com/spf13/cobra"
)

func NewSystemCmd(httpClient *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "Manage system admins",
	}

	cmd.AddCommand(NewSystemInitCmd(httpClient))
	cmd.AddCommand(NewAddAdminCmd(httpClient))

	return cmd
}
