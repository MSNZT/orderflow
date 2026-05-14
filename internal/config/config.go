package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPServer HTTPServerConfig `yaml:"http_server"`
}

type HTTPServerConfig struct {
	Addr            string
	Timeout         time.Duration
	HeaderTimeout   time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	addr := getEnv("HTTP_ADDR", ":8080")

	timeout, err := getTimeDurationEnv("HTTP_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	headerTimeout, err := getTimeDurationEnv("HTTP_HEADER_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := getTimeDurationEnv("HTTP_HEADER_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := getTimeDurationEnv("HTTP_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := getTimeDurationEnv("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPServer: HTTPServerConfig{
			Addr:            addr,
			Timeout:         timeout,
			HeaderTimeout:   headerTimeout,
			WriteTimeout:    writeTimeout,
			IdleTimeout:     idleTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
	}, nil
}

func getEnv(env string, defaultValue string) string {
	v := os.Getenv(env)
	if v == "" {
		return defaultValue
	}
	return v
}

func getTimeDurationEnv(env string, duration time.Duration) (time.Duration, error) {
	v := os.Getenv(env)
	if v == "" {
		return duration, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return duration, fmt.Errorf("failed to parse duration env: %v, err: %w", env, err)
	}

	return d, nil
}
