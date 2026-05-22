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
	Log        LogConfig   `yaml:"log"`
	Storage    Storage     `yaml:"storage"`
	Redis      RedisConfig `yaml:"redis"`
	JWT        JWT         `yaml:"jwt"`
	HashCost   int         `yaml:"hash_cost" env-default:"10"`
}

type GRPCServer struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type LogConfig struct {
	Level  string `yaml:"level"  env-default:"info"`
	Output string `yaml:"output" env-default:"stdout"`
	File   string `yaml:"file"`
}

type Storage struct {
	Type     string     `yaml:"type"`
	Postgres PostgresDB `yaml:"postgres"`
	Mongo    MongoDB    `yaml:"mongodb"`
}

type PostgresDB struct {
	Host           string        `yaml:"host"`
	Port           string        `yaml:"port"`
	User           string        `env:"DATABASE_USER"`
	Password       string        `env:"DATABASE_PASSWORD"`
	Name           string        `yaml:"name"`
	SSLMode        string        `yaml:"ssl_mode"`
	ConnectTimeout time.Duration `yaml:"connect_timeout" env-default:"5s"`
	MaxRetriesTime time.Duration `yaml:"max_retries_time" env-default:"30s"`
}

type MongoDB struct {
	URI            string        `yaml:"uri"`
	Name           string        `yaml:"name"`
	ConnectTimeout time.Duration `yaml:"connect_timeout" env-default:"5s"`
}

func (d PostgresDB) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	Password     string        `env:"REDIS_PASSWORD"`
	DB           int           `yaml:"db" env-default:"0"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env-default:"1s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env-default:"1s"`
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWT struct {
	PrivateKeyPath string        `env:"JWT_PRIVATE_KEY_PATH" env-required:"true"`
	TTL            time.Duration `yaml:"ttl" env-default:"24h"`
}

func (j JWT) GetPrivateKeyPEM() ([]byte, error) {
	return os.ReadFile(j.PrivateKeyPath)
}

func MustLoad() *Config {
	_ = godotenv.Load()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
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
