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

	logg, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logg.Sync()

	logger.InitLogger(logg)

	sugar := *logg.Sugar()
	sugar.Infow("Starting server", "addr", config.Host)

	consumer, err := storage.NewConsumer(config.FilePath)
	if err != nil {
		panic(err)
	}
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
		storage.Add(event.ShortURL, event.OriginalURL)
	}

	r.Use(logger.WithLogging)
	r.Use(compress.GzipHandle)

	r.Get("/{id}", handlers.GetHandler)
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(w, r, config)
	})
	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		handlers.JSONHandler(w, r, config)
	})

	log.Fatal(http.ListenAndServe(config.Host, r))
}
