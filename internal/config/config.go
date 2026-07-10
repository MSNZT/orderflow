package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTP     HTTPConfig     `yaml:"http"`
	Postgres PostgresConfig `yaml:"postgres"`
	JWT      JWTConfig      `yaml:"jwt"`
	Orders   OrdersConfig   `yaml:"orders"`
	Yookassa YookassaConfig `yaml:"yookassa"`
}

type HTTPConfig struct {
	Addr              string        `yaml:"addr" env:"HTTP_ADDR"`
	ReadTimeout       time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout" env:"HTTP_READ_HEADER_TIMEOUT"`
	WriteTimeout      time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT"`
	IdleTimeout       time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT"`
}

type PostgresConfig struct {
	DSN             string        `yaml:"dsn" env:"POSTGRES_DSN"`
	MaxConns        int32         `yaml:"max_conns" env:"POSTGRES_MAX_CONNS"`
	MinConns        int32         `yaml:"min_conns" env:"POSTGRES_MIN_CONNS"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime" env:"POSTGRES_MAX_CONN_LIFETIME"`
}

type JWTConfig struct {
	Secret     string        `yaml:"secret" env:"JWT_SECRET"`
	AccessTTL  time.Duration `yaml:"access_ttl" env:"JWT_ACCESS_TTL"`
	RefreshTTL time.Duration `yaml:"refresh_ttl" env:"JWT_REFRESH_TTL"`
}

type OrdersConfig struct {
	PaymentTTL       time.Duration `yaml:"payment_ttl" env:"PAYMENT_TTL"`
	ExpireInterval   time.Duration `yaml:"expire_interval" env:"EXPIRE_INTERVAL"`
	ExpireBatchLimit int           `yaml:"expire_batch_limit" env:"EXPIRE_BATCH_LIMIT"`
}

type YookassaConfig struct {
	APIURL         string        `yaml:"api_url" env:"YOOKASSA_API_URL"`
	ShopID         string        `yaml:"shop_id" env:"YOOKASSA_SHOP_ID"`
	SecretKey      string        `yaml:"secret_key" env:"YOOKASSA_SECRET_KEY"`
	ReturnURL      string        `yaml:"return_url" env:"YOOKASSA_RETURN_URL"`
	RequestTimeout time.Duration `yaml:"request_timeout" env:"YOOKASSA_REQUEST_TIMEOUT"`
}

func Load() (*Config, error) {
	CONFIG_PATH := os.Getenv("CONFIG_PATH")
	if CONFIG_PATH == "" {
		return nil, fmt.Errorf("CONFIG_PATH is not set")
	}

	var cfg Config

	if err := cleanenv.ReadConfig(CONFIG_PATH, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.HTTP.Addr == "" {
		return fmt.Errorf("http addr is required")
	}

	if c.HTTP.ReadTimeout <= 0 {
		return fmt.Errorf("http read timeout must be greater than 0")
	}

	if c.HTTP.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("http read header timeout must be greater than 0 seconds")
	}

	if c.HTTP.ShutdownTimeout <= 0*time.Second {
		return fmt.Errorf("http shutdown timeout must be greater than 0 seconds")
	}

	if c.HTTP.IdleTimeout <= 0*time.Second {
		return fmt.Errorf("http idle timeout must be greater than 0 seconds")
	}

	if c.HTTP.WriteTimeout <= 0*time.Second {
		return fmt.Errorf("http write timeout must be greater than 0 seconds")
	}

	if c.Postgres.DSN == "" {
		return fmt.Errorf("postgres dsn is required")
	}

	if c.Postgres.MaxConns <= 0 {
		return fmt.Errorf("postgres max conns must be greater than 0")
	}

	if c.Postgres.MinConns < 0 {
		return fmt.Errorf("postgres min conns must be greater than or equal to 0")
	}

	if c.Postgres.MaxConnLifetime < 0 {
		return fmt.Errorf("postgres max conn lifetime must be greater than or equal to 0")
	}

	if c.Postgres.MinConns > c.Postgres.MaxConns {
		return fmt.Errorf("postgres min conns cannot be greater than max conns")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt secret is required")
	}

	if c.JWT.AccessTTL <= 0 {
		return fmt.Errorf("jwt access ttl must be greater than 0")
	}

	if c.JWT.RefreshTTL <= 0 {
		return fmt.Errorf("jwt refresh ttl must be greater than 0")
	}

	if c.Orders.PaymentTTL <= 0 {
		return fmt.Errorf("orders payment ttl must be greater than 0")
	}

	if c.Orders.ExpireInterval <= 0 {
		return fmt.Errorf("orders expire interval must be greater than 0")
	}

	if c.Orders.ExpireBatchLimit < 100 {
		return fmt.Errorf("orders expire batch limit must be greater than or equal to 100")
	}

	if err := c.Yookassa.validate(); err != nil {
		return err
	}

	return nil
}

func (c *YookassaConfig) validate() error {
	if strings.TrimSpace(c.APIURL) == "" {
		return fmt.Errorf("yookassa api url is required")
	}

	if strings.TrimSpace(c.ShopID) == "" {
		return fmt.Errorf("yookassa shop id is required")
	}

	if strings.TrimSpace(c.SecretKey) == "" {
		return fmt.Errorf("yookassa secret key is required")
	}

	if strings.TrimSpace(c.ReturnURL) == "" {
		return fmt.Errorf("yookassa return url is required")
	}

	if c.RequestTimeout <= 0 {
		return fmt.Errorf("yookassa request timeout must be greater than 0")
	}

	return nil
}
