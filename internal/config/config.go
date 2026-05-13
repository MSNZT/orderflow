package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTPServer `yaml:"http_server"`
}

type HTTPServer struct {
	Addr            string        `yaml:"addr" env-default:"localhost:8080"`
	Timeout         time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"30s"`
}

func MustLoad() Config {
	CONFIG_PATH := os.Getenv("CONFIG_PATH")
	if CONFIG_PATH == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(CONFIG_PATH); err != nil {
		log.Fatalf("Config file doesn't exists %s", CONFIG_PATH)
	}

	var config Config

	if err := cleanenv.ReadConfig(CONFIG_PATH, &config); err != nil {
		log.Fatalf("Cannot read config file: %s", err)
	}

	return config
}
