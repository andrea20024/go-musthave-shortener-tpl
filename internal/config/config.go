// Package config provides application configuration initialization and
// validation utilities.
//
// Configuration is loaded from the following sources in order:
//   - Environment variables (via github.com/caarlos0/env/v6)
//   - Command-line flags (overriding environment defaults)
//   - Hardcoded defaults
//
// Supported environment variables:
//
//	SERVER_ADDRESS   — server listen address (default: localhost:8080)
//	BASE_URL         — base URL for generated short links (default: http://localhost:8080)
//	FILE_STORAGE_PATH — path to the file-based storage (default: storage.json)
//	DATABASE_DSN     — PostgreSQL connection string (PostgreSQL mode)
//	AUTH_SECRET      — secret key for user cookie signing (required)
//	WORKER_BUFFER_SIZE — size of the async delete worker queue (default: 100)
//	AUDIT_FILE       — path for file-based audit log
//	AUDIT_URL        — URL for HTTP-based audit log
package config

import "fmt"

// Config holds the application configuration loaded from environment
// variables and command-line flags.
// generate:reset
type Config struct {
	// Host is the network address the HTTP server listens on.
	// Environment variable: SERVER_ADDRESS
	Host string `env:"SERVER_ADDRESS"`

	// BaseURL is the base URL used to construct full short links.
	// Environment variable: BASE_URL
	BaseURL string `env:"BASE_URL"`

	// FilePath is the path to the JSON file used by FileRepository.
	// When set and database is unavailable, file-based storage is used.
	// Environment variable: FILE_STORAGE_PATH
	FilePath string `env:"FILE_STORAGE_PATH"`

	// DB is the PostgreSQL data source name (DSN) used by the database-backed
	// repository.
	// Environment variable: DATABASE_DSN
	DB string `env:"DATABASE_DSN"`

	// AuthSecret is the secret key used to sign user identification cookies.
	// This field is required; validation fails if empty.
	// Environment variable: AUTH_SECRET
	AuthSecret string `env:"AUTH_SECRET"`

	// WorkerBufferSize is the capacity of the buffered channel used by the
	// async delete Worker.
	// Environment variable: WORKER_BUFFER_SIZE
	WorkerBufferSize int `env:"WORKER_BUFFER_SIZE"`

	// AuditFile is the path to the file used as an audit log receiver.
	// Environment variable: AUDIT_FILE
	AuditFile string `env:"AUDIT_FILE"`

	// AuditURL is the HTTP URL used as an audit log receiver.
	// Environment variable: AUDIT_URL
	AuditURL string `env:"AUDIT_URL"`

	// StoreType indicates which storage backend is active ("memory", "file", or "db").
	StoreType string
}

// InitConfig returns a Config instance populated with default values.
// These defaults can be overridden by environment variables or command-line
// flags parsed via flag.Parse().
func InitConfig() *Config {
	return &Config{
		Host:             "localhost:8080",
		BaseURL:          "http://localhost:8080",
		FilePath:         "storage.json",
		DB:               "host=localhost port=5434 user=loader password=1234 dbname=truecode_db sslmode=disable",
		WorkerBufferSize: 100,
		AuditFile:        "audit.txt",
		AuditURL:         "http://localhost:5001/audit",
		StoreType:        "memory",
	}
}

// Validate checks the configuration for required fields and returns an error
// if validation fails. Currently it only checks that AuthSecret is non-empty.
func Validate(cfg *Config) error {
	if cfg.AuthSecret == "" {
		return fmt.Errorf("AUTH_SECRET is required")
	}
	return nil
}
