// Package main is the entry point for the URL Shortener service.
//
// It initializes the configuration via config.Load() and starts the HTTP
// server with all configured middleware and routes.
//
// Usage:
//
//	go run ./cmd/shortener [flags]
//
// All configuration options (flags, environment variables, JSON config file)
// are documented in the config package.
package main

import (
	"fmt"
	"log"

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

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
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
