package server

import (
	"io"
	"log"
	"net/http"

	compress "github.com/andrea20024/go-musthave-shortener-tpl/internal/compress"
	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	handlers "github.com/andrea20024/go-musthave-shortener-tpl/internal/handler"
	logger "github.com/andrea20024/go-musthave-shortener-tpl/internal/logger"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func Start(config *config.Config) {
	r := chi.NewRouter()

	var repo storage.Repository

	repo = storage.NewMapRepository()

	if config.FilePath != "" {
		newRepo, err := storage.NewFileRepository(config.FilePath)
		if err == nil {
			repo = newRepo
		}
	}

	/*
		if config.DB != "" {
			newRepo := storage.Init(config.DB)
			if newRepo != nil {
				repo = newRepo
			}
		}
	*/

	logg, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logg.Sync()

	logger.InitLogger(logg)

	sugar := *logg.Sugar()
	sugar.Infow("Starting server", "addr", config.Host)

	if config.FilePath != "" {
		consumer, err := storage.NewConsumer(config.FilePath)
		if err != nil {
			log.Printf("NewConsumer error: %v", err)
		} else {
			for {
				event, err := consumer.ReadEvent()
				if err != nil {
					if err == io.EOF {
						break
					}
					continue
				}
				if event == nil {
					break
				}
				repo.Add(event.ShortURL, event.OriginalURL)
			}
		}
	}

	r.Use(logger.WithLogging)
	r.Use(compress.GzipHandle)

	r.Get("/{id}", func(w http.ResponseWriter, req *http.Request) {
		handlers.GetHandler(w, req, repo)
	})
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(w, r, config, repo)
	})
	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		handlers.JSONHandler(w, r, config, repo)
	})
	r.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
		handlers.PingHandler(w, req, repo)
	})

	log.Fatal(http.ListenAndServe(config.Host, r))
}
