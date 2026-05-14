package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPServer HTTPServerConfig
}

type HTTPServerConfig struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
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

	return Config{
		HTTPServer: HTTPServerConfig{
			Addr:              addr,
			ReadTimeout:       readTimeout,
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			ShutdownTimeout:   shutdownTimeout,
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

func getDurationEnv(key string, duration time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return duration, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return duration, fmt.Errorf("invalid duration env %s: %w, err: %w", key, err)
	}

	return d, nil
}
