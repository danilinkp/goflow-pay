package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env        string      `yaml:"env" env:"ENV" env-default:"local"`
	GRPCServer GRPCServer  `yaml:"grpc"`
	DB         DB          `yaml:"db"`
	Redis      RedisConfig `yaml:"redis"`
	JWT        JWT         `yaml:"jwt"`
	HashCost   int         `yaml:"hash_cost" env:"HASH_COST" env-default:"10"`
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

func (d DB) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Host         string        `yaml:"host" env:"REDIS_HOST"`
	Port         int           `yaml:"port" env:"REDIS_PORT"`
	Password     string        `yaml:"password" env:"REDIS_PASSWORD"`
	DB           int           `yaml:"db" env-default:"0"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env-default:"1s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env-default:"1s"`
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWT struct {
	PrivateKeyPath string        `yaml:"private_key_path" env:"JWT_PRIVATE_KEY_PATH" env-required:"true"`
	TTL            time.Duration `yaml:"ttl" env:"JWT_TTL" env-default:"24h"`
}

func (j JWT) GetPrivateKeyPEM() ([]byte, error) {
	return os.ReadFile(j.PrivateKeyPath)
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

	if cfg.HashCost < 4 || cfg.HashCost > 31 {
		log.Fatalf("invalid hash_cost: %d (must be 4-31)", cfg.HashCost)
	}

	return &cfg
}
