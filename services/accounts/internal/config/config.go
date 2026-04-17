package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env         string      `yaml:"env" env:"ENV" env-default:"local"`
	GRPCServer  GRPCServer  `yaml:"grpc"`
	DB          DB          `yaml:"db"`
	Kafka       Kafka       `yaml:"kafka"`
	Outbox      Outbox      `yaml:"outbox"`
	BankGateway BankGateway `yaml:"bank_gateway"`
}

type GRPCServer struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type DB struct {
	Host           string        `env:"DATABASE_HOST" yaml:"host"`
	Port           string        `env:"DATABASE_PORT" yaml:"port"`
	User           string        `env:"DATABASE_USER" yaml:"user"`
	Password       string        `env:"DATABASE_PASSWORD" yaml:"password"`
	Name           string        `env:"DATABASE_NAME" yaml:"name"`
	SSLMode        string        `env:"DATABASE_SSL_MODE" yaml:"ssl_mode"`
	ConnectTimeout time.Duration `env:"DATABASE_CONNECTION_TIMEOUT" yaml:"connect_timeout" env-default:"5s"`
	MaxRetriesTime time.Duration `env:"DATABASE_MAX_RETRIES_TIME" yaml:"max_retries_time" env-default:"30s"`
}

type Kafka struct {
	BrokersRaw   string        `env:"KAFKA_BROKERS" yaml:"brokers" env-required:"true"`
	WriteTimeout time.Duration `env:"KAFKA_WRITE_TIMEOUT" yaml:"write_timeout" env-default:"5s"`
}

func (k Kafka) Brokers() []string {
	return strings.Split(k.BrokersRaw, ",")
}

type Outbox struct {
	Interval  time.Duration `enc:"OUTBOX_INTERVAL" yaml:"interval" env-default:"5s"`
	BatchSize int           `yaml:"batch_size" env-default:"10"`
}

type BankGateway struct {
	FailureRate float64 `yaml:"failure_rate" env-default:"0.1"`
}

func (d DB) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
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
