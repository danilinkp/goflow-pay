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

	return &cfg, nil
}
