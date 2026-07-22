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
	flag.StringVar(&config.FilePath, "f", config.FilePath, "file path")
	flag.StringVar(&config.DB, "d", config.DB, "database")
	flag.Parse()
	
	if config.BaseURL == "" {
		config.BaseURL = "http://" + config.Host
	}
	
	hostFlag := flag.Lookup("a").Value.String()
	baseURLFlag := flag.Lookup("b").Value.String()
	
	env.Parse(config)
	
	if hostFlag != config.Host {
		config.Host = hostFlag
	}
	if baseURLFlag != config.BaseURL {
		config.BaseURL = baseURLFlag
	}

	if config.BaseURL == "" {
		config.BaseURL = "http://" + config.Host
	}

	server.Start(config)
}
