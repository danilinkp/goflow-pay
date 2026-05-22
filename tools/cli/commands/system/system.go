package system

import (
	"cli/internal/client"
	"cli/internal/config"

	"github.com/spf13/cobra"
)

func NewSystemCmd(httpClient *client.Client, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "Manage system admins",
	}

	cmd.AddCommand(NewSystemInitCmd(httpClient))
	cmd.AddCommand(NewAddAdminCmd(httpClient))
	cmd.AddCommand(NewMigrateToMongoCmd(cfg))

	return cmd
}
