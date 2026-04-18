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
	Env      string     `yaml:"env" env:"ENV" env-default:"local"`
	DB       DB         `yaml:"db"`
	Kafka    Kafka      `yaml:"kafka"`
	SMTP     SMTPConfig `yaml:"smtp"`
	AuthGRPC GRPCConfig `yaml:"auth_grpc"`
}

type DB struct {
	Host           string        `yaml:"host"`
	Port           string        `yaml:"port"`
	User           string        `env:"DATABASE_USER"`
	Password       string        `env:"DATABASE_PASSWORD"`
	Name           string        `yaml:"name"`
	SSLMode        string        `yaml:"ssl_mode"`
	ConnectTimeout time.Duration `yaml:"connect_timeout" env-default:"5s"`
	MaxRetriesTime time.Duration `yaml:"max_retries_time" env-default:"30s"`
}

type Kafka struct {
	BrokersRaw string `yaml:"brokers" env-required:"true"`
	Topic      string `yaml:"topic" env-default:"5s"`
}

func (k Kafka) Brokers() []string {
	return strings.Split(k.BrokersRaw, ",")
}

type SMTPConfig struct {
	Host     string `env:"SMTP_HOST"`
	Port     int    `env:"SMTP_PORT" env-default:"587"`
	Password string `env:"SMTP_PASSWORD"`
	From     string `env:"SMTP_FROM"`
}

type GRPCConfig struct {
	Addr string `yaml:"addr" env:"AUTH_GRPC_ADDR"`
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
