// Package main is the entry point for the URL Shortener service.
//
// It initializes the configuration from environment variables and command-line
// flags, then starts the HTTP server with all configured middleware and routes.
//
// Usage:
//
//	go run ./cmd/shortener [flags]
//
// Flags:
//
//	-a string        host address (default "localhost:8080")
//	-b string        base URL for short links (default "http://localhost:8080")
//	-f string        file storage path (optional)
//	-d string        PostgreSQL DSN (optional)
//	-worker-buffer int worker channel buffer size (default 100)
//	-audit-file      file path for audit log
//	-audit-url       HTTP URL for audit log
//	-s               enable HTTPS
//	-tls-cert        path to TLS certificate file
//	-tls-key         path to TLS key file
//	-c, -config      path to JSON config file
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/caarlos0/env/v6"

	"github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	"github.com/andrea20024/go-musthave-shortener-tpl/internal/server"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()
	cfg := config.InitConfig()

	configFile := flag.String("c", "", "path to JSON config file")
	configFileAlt := flag.String("config", "", "path to JSON config file")
	flag.StringVar(&cfg.Host, "a", cfg.Host, "host")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "base url")
	flag.StringVar(&cfg.FilePath, "f", cfg.FilePath, "file path")
	flag.StringVar(&cfg.DB, "d", cfg.DB, "database")
	flag.IntVar(&cfg.WorkerBufferSize, "worker-buffer", cfg.WorkerBufferSize, "worker buffer size")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "audit file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "audit URL")
	flag.StringVar(&cfg.TLSCertFile, "tls-cert", cfg.TLSCertFile, "TLS certificate file path")
	flag.StringVar(&cfg.TLSKeyFile, "tls-key", cfg.TLSKeyFile, "TLS key file path")
	flag.BoolVar(&cfg.EnableHTTPS, "s", cfg.EnableHTTPS, "enable HTTPS")
	flag.Parse()

	// Determine config file path from flags or env var CONFIG
	cfgFile := *configFile
	if cfgFile == "" {
		cfgFile = *configFileAlt
	}
	if cfgFile == "" {
		cfgFile = os.Getenv("CONFIG")
	}

	// Load JSON config (lowest priority, before flags and env vars)
	// Flags and env vars will override JSON values
	if cfgFile != "" {
		// Save flag-overridden values
		flagValues := map[string]string{
			"Host":        cfg.Host,
			"BaseURL":     cfg.BaseURL,
			"FilePath":    cfg.FilePath,
			"DB":          cfg.DB,
			"AuditFile":   cfg.AuditFile,
			"AuditURL":    cfg.AuditURL,
			"TLSCertFile": cfg.TLSCertFile,
			"TLSKeyFile":  cfg.TLSKeyFile,
		}
		flagInts := map[string]int{
			"WorkerBufferSize": cfg.WorkerBufferSize,
		}
		flagBools := map[string]bool{
			"EnableHTTPS": cfg.EnableHTTPS,
		}

		// Reset to defaults and load JSON
		*cfg = *config.InitConfig()

		if err := config.LoadConfigFromJSON(cfg, cfgFile); err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}

		// Restore flag-overridden values (flags have highest priority)
		if flagValues["Host"] != "" {
			cfg.Host = flagValues["Host"]
		}
		if flagValues["BaseURL"] != "" {
			cfg.BaseURL = flagValues["BaseURL"]
		}
		if flagValues["FilePath"] != "" {
			cfg.FilePath = flagValues["FilePath"]
		}
		if flagValues["DB"] != "" {
			cfg.DB = flagValues["DB"]
		}
		if flagValues["AuditFile"] != "" {
			cfg.AuditFile = flagValues["AuditFile"]
		}
		if flagValues["AuditURL"] != "" {
			cfg.AuditURL = flagValues["AuditURL"]
		}
		if flagInts["WorkerBufferSize"] > 0 {
			cfg.WorkerBufferSize = flagInts["WorkerBufferSize"]
		}
		if flagBools["EnableHTTPS"] {
			cfg.EnableHTTPS = true
		}
	}

	env.Parse(cfg)

	if err := config.Validate(cfg); err != nil {
		//Можно и остановку поставить - log.Fatal(err)
		log.Printf("Config validation error: %v", err)
	}

	server.Start(cfg)
}

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", versionOrDefault(buildVersion))
	fmt.Printf("Build date: %s\n", versionOrDefault(buildDate))
	fmt.Printf("Build commit: %s\n", versionOrDefault(buildCommit))
}

func versionOrDefault(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}
