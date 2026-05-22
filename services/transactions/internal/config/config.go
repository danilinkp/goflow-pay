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
	Env        string     `yaml:"env" env:"ENV" env-default:"local"`
	Log        LogConfig  `yaml:"log"`
	GRPCServer GRPCServer `yaml:"grpc"`
	GRPCClient GRPCClient `yaml:"grpc_client"`
	Storage    Storage    `yaml:"storage"`
	Kafka      Kafka      `yaml:"kafka"`
	Outbox     Outbox     `yaml:"outbox"`
	Recover    Recover    `yaml:"recover"`
}

type LogConfig struct {
	Level  string `yaml:"level"  env-default:"info"`
	Output string `yaml:"output" env-default:"stdout"`
	File   string `yaml:"file"`
}

type GRPCServer struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type GRPCClient struct {
	Addr string `yaml:"addr"`
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

type Kafka struct {
	BrokersRaw   string        `yaml:"brokers" env-required:"true"`
	WriteTimeout time.Duration `yaml:"write_timeout" env-default:"5s"`
}

func (k Kafka) Brokers() []string {
	return strings.Split(k.BrokersRaw, ",")
}

type Outbox struct {
	Interval  time.Duration `yaml:"interval" env-default:"5s"`
	BatchSize int           `yaml:"batch_size" env-default:"10"`
}

type Recover struct {
	Interval   time.Duration `yaml:"interval" env-default:"5s"`
	StaleAfter time.Duration `yaml:"stale_after" env-default:"30s"`
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
