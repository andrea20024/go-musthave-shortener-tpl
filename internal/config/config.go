package config

import "fmt"

type Config struct {
	Host             string `env:"SERVER_ADDRESS"`
	BaseURL          string `env:"BASE_URL"`
	FilePath         string `env:"FILE_STORAGE_PATH"`
	DB               string `env:"DATABASE_DSN"`
	AuthSecret       string `env:"AUTH_SECRET"`
	WorkerBufferSize int    `env:"WORKER_BUFFER_SIZE"`
	StoreType        string
}

func InitConfig() *Config {
	return &Config{
		Host:             "localhost:8080",
		BaseURL:          "http://localhost:8080",
		FilePath:         "storage.json",
		DB:               "host=localhost port=5434 user=loader password=1234 dbname=truecode_db sslmode=disable",
		WorkerBufferSize: 100,
		StoreType:        "memory",
	}
}

func Validate(cfg *Config) error {
	if cfg.AuthSecret == "" {
		return fmt.Errorf("AUTH_SECRET is required")
	}
	return nil
}
