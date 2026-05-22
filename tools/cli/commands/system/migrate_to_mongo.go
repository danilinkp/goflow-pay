package system

import (
	"cli/internal/config"
	"cli/internal/migration"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewMigrateToMongoCmd(cfg *config.Config) *cobra.Command {
	var service string
	var batchSize int

	cmd := &cobra.Command{
		Use:   "migrate-to-mongo",
		Short: "Migrate data from PostgreSQL to MongoDB",
		Long:  "Migrates all data from PostgreSQL to MongoDB for specified service (or all services)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			services := map[string]config.ServiceMigrationConfig{
				"accounts":      cfg.Migration.Accounts,
				"transactions":  cfg.Migration.Transactions,
				"auth":          cfg.Migration.Auth,
				"notifications": cfg.Migration.Notifications,
			}

			if service != "all" {
				svcCfg, ok := services[service]
				if !ok {
					return fmt.Errorf("unknown service %q, available: accounts, transactions, auth, notifications, all", service)
				}
				return runMigration(ctx, service, svcCfg, batchSize)
			}

			for name, svcCfg := range services {
				fmt.Printf("migrating %s...\n", name)
				if err := runMigration(ctx, name, svcCfg, batchSize); err != nil {
					return fmt.Errorf("migration failed for %s: %w", name, err)
				}
				fmt.Printf("%s done\n", name)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&service, "service", "all", "Service to migrate (accounts|transactions|auth|notifications|all)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 1000, "Number of documents per batch")

	return cmd
}

func runMigration(ctx context.Context, name string, cfg config.ServiceMigrationConfig, batchSize int) error {
	switch name {
	case "accounts":
		return migration.MigrateAccounts(ctx, cfg.PostgresDSN, cfg.MongoDSN, cfg.MongoDBName, batchSize)
	case "transactions":
		return migration.MigrateTransactions(ctx, cfg.PostgresDSN, cfg.MongoDSN, cfg.MongoDBName, batchSize)
	case "auth":
		return migration.MigrateAuth(ctx, cfg.PostgresDSN, cfg.MongoDSN, cfg.MongoDBName, batchSize)
	case "notifications":
		return migration.MigrateNotifications(ctx, cfg.PostgresDSN, cfg.MongoDSN, cfg.MongoDBName, batchSize)
	default:
		return fmt.Errorf("unknown service: %s", name)
	}
}
