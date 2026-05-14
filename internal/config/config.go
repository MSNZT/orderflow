package config

import (
	"log"
	"os"
	"time"
)

type Config struct {
	HTTPServer HTTPServerConfig `yaml:"http_server"`
}

type HTTPServerConfig struct {
	Addr            string        `yaml:"addr" env-default:":8080"`
	Timeout         time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"30s"`
}

func Load() Config {
	return Config{
		HTTPServer: HTTPServerConfig{
			Addr:            getEnv("HTTP_ADDR", ":8080"),
			Timeout:         getTimeDurationEnv("HTTP_TIMEOUT", 4*time.Second),
			IdleTimeout:     getTimeDurationEnv("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getTimeDurationEnv("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
	}
}

func getEnv(env string, defaultValue string) string {
	v := os.Getenv(env)
	if v == "" {
		return defaultValue
	}
	return v
}

func getTimeDurationEnv(env string, defaultValue time.Duration) time.Duration {
	v := os.Getenv(env)
	if v == "" {
		return defaultValue
	}

	if d, err := time.ParseDuration(v); err == nil {
		return d
	} else {
		log.Printf("Failed to parse env: %v, err: %v", env, err)
	}

	return defaultValue
}
