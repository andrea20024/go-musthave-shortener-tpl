package main

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"

	"github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	"github.com/andrea20024/go-musthave-shortener-tpl/internal/server"
)

func main() {
	cfg := config.InitConfig()

	flag.StringVar(&cfg.Host, "a", cfg.Host, "host")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "base url")
	flag.StringVar(&cfg.FilePath, "f", cfg.FilePath, "file path")
	flag.StringVar(&cfg.DB, "d", cfg.DB, "database")
	flag.IntVar(&cfg.WorkerBufferSize, "worker-buffer", cfg.WorkerBufferSize, "worker buffer size")
	flag.Parse()
	env.Parse(cfg)

	if err := config.Validate(cfg); err != nil {
		//Можно и остановку поставить - log.Fatal(err)
		log.Printf("Config validation error: %v", err)
	}

	server.Start(cfg)
}
