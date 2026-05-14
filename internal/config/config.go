package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPServer HTTPServerConfig
	DB         DBConfig
}

type HTTPServerConfig struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type DBConfig struct {
	DSN             string
	MaxOpenConns    int32
	MaxIddleConns   int32
	ConnMaxLifetime time.Duration
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

	connMaxLifetime, err := getDurationEnv("POSTGRES_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}

	maxOpenConns, err := getIntEnv("POSTGRES_MAX_OPEN_CONNS", 10)
	if err != nil {
		return Config{}, err
	}

	maxIddleConns, err := getIntEnv("POSTGRES_MAX_IDLE_CONNS", 5)
	if err != nil {
		return Config{}, err
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
		DB: DBConfig{
			DSN:             dsn,
			ConnMaxLifetime: connMaxLifetime,
			MaxOpenConns:    int32(maxOpenConns),
			MaxIddleConns:   int32(maxIddleConns),
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
		return 0, fmt.Errorf("invalid convert to int: %w", err)
	}

	return n, nil
}
