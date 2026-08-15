package main

import (
	"flag"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

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
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "audit file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "audit URL")
	flag.Parse()

	if err := config.Validate(cfg); err != nil {
		log.Printf("Config validation error: %v", err)
	}

	go server.Start(cfg)

	fmt.Println("Profiler server started on " + cfg.Host)
	fmt.Println("pprof available at http://" + cfg.Host + "/debug/pprof/")
	fmt.Println("Press Ctrl+C to stop")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
