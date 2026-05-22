package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	GatewayURL string `env:"GATEWAY_URL" env-default:"http://localhost:8080"`

	Migration MigrationConfig
}

type MigrationConfig struct {
	Accounts      ServiceMigrationConfig
	Transactions  ServiceMigrationConfig
	Auth          ServiceMigrationConfig
	Notifications ServiceMigrationConfig
}

type ServiceMigrationConfig struct {
	PostgresDSN string
	MongoDSN    string
	MongoDBName string
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err == nil {
		_ = godotenv.Load(filepath.Join(home, ".goflow", ".env"))
	}

	_ = godotenv.Load()

	var cfg Config
	if err = cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg.Migration = MigrationConfig{
		Accounts: ServiceMigrationConfig{
			PostgresDSN: getEnv("ACCOUNTS_POSTGRES_DSN", "postgres://postgres:postgres@localhost:5433/account_service?sslmode=disable"),
			MongoDSN:    getEnv("ACCOUNTS_MONGO_DSN", "mongodb://localhost:27018/?directConnection=true"),
			MongoDBName: getEnv("ACCOUNTS_MONGO_DB", "accounts_service"),
		},
		Transactions: ServiceMigrationConfig{
			PostgresDSN: getEnv("TRANSACTIONS_POSTGRES_DSN", "postgres://postgres:postgres@localhost:5434/transaction_service?sslmode=disable"),
			MongoDSN:    getEnv("TRANSACTIONS_MONGO_DSN", "mongodb://localhost:27019/?directConnection=true"),
			MongoDBName: getEnv("TRANSACTIONS_MONGO_DB", "transactions_service"),
		},
		Auth: ServiceMigrationConfig{
			PostgresDSN: getEnv("AUTH_POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/auth_service?sslmode=disable"),
			MongoDSN:    getEnv("AUTH_MONGO_DSN", "mongodb://localhost:27017/?directConnection=true"),
			MongoDBName: getEnv("AUTH_MONGO_DB", "auth_service"),
		},
		Notifications: ServiceMigrationConfig{
			PostgresDSN: getEnv("NOTIFICATIONS_POSTGRES_DSN", "postgres://postgres:postgres@localhost:5435/notification_service?sslmode=disable"),
			MongoDSN:    getEnv("NOTIFICATIONS_MONGO_DSN", "mongodb://localhost:27020/?directConnection=true"),
			MongoDBName: getEnv("NOTIFICATIONS_MONGO_DB", "notifications_service"),
		},
	}

	return &cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
