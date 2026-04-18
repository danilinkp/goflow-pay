package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env              string      `yaml:"env" env-default:"local"`
	HTTPServer       HTTPServer  `yaml:"http_server"`
	GRPCClients      GRPCClients `yaml:"grpc_clients"`
	JWTPublicKeyPath string      `env:"JWT_PUBLIC_KEY_PATH"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"0.0.0.0:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type GRPCClients struct {
	AuthAddress        string `yaml:"auth_address" env-default:"auth_service:50051"`
	AccountAddress     string `yaml:"account_address" env-default:"account_service:50051"`
	TransactionAddress string `yaml:"transaction_address" env-default:"transaction_service:50051"`
}

func MustLoad() *Config {
	_ = godotenv.Load()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/local.yaml"
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file %s does not exist", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %v", err)
	}

	return &cfg
}
