package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPServer HTTPServerConfig
	Postgres   PostgresConfig
}

type HTTPServerConfig struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type PostgresConfig struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

func Load() (Config, error) {
	addr := getEnv("HTTP_ADDR", ":8080")

	readTimeout, err := getDurationEnv("HTTP_READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	readHeaderTimeout, err := getDurationEnv("HTTP_READ_HEADER_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := getDurationEnv("HTTP_WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := getDurationEnv("HTTP_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := getDurationEnv("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	dsn := getEnv("POSTGRES_DSN", "postgres://orderflow:orderflow@localhost:5432/orderflow?sslmode=disable")

	maxConnLifetime, err := getDurationEnv("POSTGRES_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}

	maxConns, err := getIntEnv("POSTGRES_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}

	if maxConns < 0 {
		return Config{}, fmt.Errorf("max conns must be greater than 0: %v", maxConns)
	}

	minConns, err := getIntEnv("POSTGRES_MIN_CONNS", 0)
	if err != nil {
		return Config{}, err
	}

	if minConns <= 0 || minConns <= maxConns {
		return Config{}, fmt.Errorf("max conns must be less than or equal maxConns: %v", maxConns)
	}

	return Config{
		HTTPServer: HTTPServerConfig{
			Addr:              addr,
			ReadTimeout:       readTimeout,
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			ShutdownTimeout:   shutdownTimeout,
		},
		Postgres: PostgresConfig{
			DSN:             dsn,
			MaxConnLifetime: maxConnLifetime,
			MaxConns:        int32(maxConns),
			MinConns:        int32(minConns),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v
}

func getDurationEnv(key string, duration time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return duration, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid duration env %s: %w", key, err)
	}

	return d, nil
}

func getIntEnv(key string, n int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return n, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid int env %s=%q: %w", key, v, err)
	}

	return n, nil
}
