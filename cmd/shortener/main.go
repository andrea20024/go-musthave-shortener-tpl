package main

import (
	"flag"

	"github.com/caarlos0/env/v6"

	"github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	"github.com/andrea20024/go-musthave-shortener-tpl/internal/server"
)

func main() {
	config := config.InitConfig()

	flag.StringVar(&config.Host, "a", config.Host, "host")
	flag.StringVar(&config.BaseURL, "b", config.BaseURL, "base url")
	flag.Parse()
	env.Parse(config)

	server.Start(config.Host, config.BaseURL)
}
