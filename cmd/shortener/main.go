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
package main

import (
	"flag"
	"fmt"
	"log"

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

	flag.StringVar(&cfg.Host, "a", cfg.Host, "host")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "base url")
	flag.StringVar(&cfg.FilePath, "f", cfg.FilePath, "file path")
	flag.StringVar(&cfg.DB, "d", cfg.DB, "database")
	flag.IntVar(&cfg.WorkerBufferSize, "worker-buffer", cfg.WorkerBufferSize, "worker buffer size")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "audit file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "audit URL")
	flag.Parse()
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
