package config

import (
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

func MustLoad() Config {
	return Config{
		HTTPServer: HTTPServerConfig{
			Addr:            ":8080",
			Timeout:         4 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
		},
	}
}
