// Package config provides application configuration initialization and
// validation utilities.
//
// Configuration is loaded from the following sources in order (higher priority first):
//  1. Command-line flags (highest priority)
//  2. Environment variables
//  3. JSON config file (lowest priority)
//  4. Hardcoded defaults
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
//	ENABLE_HTTPS     — enable TLS (default: false)
//	CONFIG           — path to JSON config file
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

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

	// EnableHTTPS enables TLS for the HTTP server.
	// Controlled by command-line flag "-s" or environment variable "ENABLE_HTTPS".
	EnableHTTPS bool `env:"ENABLE_HTTPS"`

	// TLSCertFile is the path to the TLS certificate file.
	// Command-line flag: -tls-cert
	TLSCertFile string

	// TLSKeyFile is the path to the TLS private key file.
	// Command-line flag: -tls-key
	TLSKeyFile string

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
		EnableHTTPS:      false,
	}
}

// Validate checks the configuration for required fields and returns an error
// if validation fails. Currently it only checks that AuthSecret is non-empty.
func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.AuthSecret == "" {
		return fmt.Errorf("AUTH_SECRET is required")
	}
	return nil
}

// fileConfig is the JSON structure for file-based configuration.
// Only fields set in InitConfig are supported.
type fileConfig struct {
	ServerAddress    string `json:"server_address"`
	BaseURL          string `json:"base_url"`
	FileStoragePath  string `json:"file_storage_path"`
	DatabaseDSN      string `json:"database_dsn"`
	WorkerBufferSize int    `json:"worker_buffer_size"`
	AuditFile        string `json:"audit_file"`
	AuditURL         string `json:"audit_url"`
	EnableHTTPS      bool   `json:"enable_https"`
	TLSCertFile      string `json:"tls_cert_file"`
	TLSKeyFile       string `json:"tls_key_file"`
}

// LoadConfigFromJSON reads configuration from a JSON file and applies it
// to the provided Config struct. Only non-empty values from the file are applied.
func LoadConfigFromJSON(cfg *Config, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config file: %w", err)
	}

	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	if fc.ServerAddress != "" {
		cfg.Host = fc.ServerAddress
	}
	if fc.BaseURL != "" {
		cfg.BaseURL = fc.BaseURL
	}
	if fc.FileStoragePath != "" {
		cfg.FilePath = fc.FileStoragePath
	}
	if fc.DatabaseDSN != "" {
		cfg.DB = fc.DatabaseDSN
	}
	if fc.WorkerBufferSize > 0 {
		cfg.WorkerBufferSize = fc.WorkerBufferSize
	}
	if fc.AuditFile != "" {
		cfg.AuditFile = fc.AuditFile
	}
	if fc.AuditURL != "" {
		cfg.AuditURL = fc.AuditURL
	}
	if fc.EnableHTTPS {
		cfg.EnableHTTPS = true
	}
	if fc.TLSCertFile != "" {
		cfg.TLSCertFile = fc.TLSCertFile
	}
	if fc.TLSKeyFile != "" {
		cfg.TLSKeyFile = fc.TLSKeyFile
	}

	return nil
}
