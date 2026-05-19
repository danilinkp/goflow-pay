package commands

import (
	"cli/commands/accounts"
	"cli/commands/auth"
	"cli/commands/companies"
	"cli/commands/system"
	"cli/commands/transactions"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"cli/internal/client"
	"cli/internal/config"
	"cli/internal/credentials"
)

var (
	cfg        *config.Config
	httpClient *client.Client
)

func NewRootCmd(cfg *config.Config) *cobra.Command {
	httpClient = client.New(cfg.GatewayURL)

	var rootCmd = &cobra.Command{
		Use:   "goflow-cli",
		Short: "GoFlow Pay administrative CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			noAuth := map[string]bool{
				"login":            true,
				"register":         true,
				"register-company": true,
				"init":             true,
			}
			if noAuth[cmd.Name()] {
				return nil
			}

			creds, err := credentials.Load()
			if err != nil {
				return fmt.Errorf("credentials load: %w", err)
			}
			httpClient.SetToken(creds.Token)
			return nil
		},
	}

	authCmd := auth.NewAuthCmd(httpClient)
	systemCmd := system.NewSystemCmd(httpClient, cfg)
	companiesCmd := companies.NewCompaniesCmd(httpClient)
	accountsCmd := accounts.NewAccountsCmd(httpClient)
	transactionsCmd := transactions.NewTransactionsCmd(httpClient)

	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(systemCmd)
	rootCmd.AddCommand(companiesCmd)
	rootCmd.AddCommand(accountsCmd)
	rootCmd.AddCommand(transactionsCmd)

	return rootCmd
}

func Execute() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	rootCmd := NewRootCmd(cfg)

	if err = rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
