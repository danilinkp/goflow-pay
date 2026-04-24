package auth

import (
	"cli/internal/client"

	"github.com/spf13/cobra"
)

func NewAuthCmd(httpClient *client.Client) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Auth commands",
	}

	authCmd.AddCommand(NewLoginCmd(httpClient))
	authCmd.AddCommand(NewRegisterNewCompanyCmd(httpClient))
	authCmd.AddCommand(NewRegisterExistingCompanyCmd(httpClient))
	authCmd.AddCommand(NewLogoutCmd(httpClient))

	return authCmd
}
