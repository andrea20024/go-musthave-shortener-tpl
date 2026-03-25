package server

import (
	"log"
	"net/http"

	handlers "github.com/andrea20024/go-musthave-shortener-tpl/internal/handler"
	logger "github.com/andrea20024/go-musthave-shortener-tpl/internal/logger"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func Start(host string, baseURL string) {
	r := chi.NewRouter()

	logg, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logg.Sync()

	logger.InitLogger(logg)

	sugar := *logg.Sugar()
	sugar.Infow("Starting server", "addr", host)
	r.Use(logger.WithLogging)

	r.Get("/{id}", handlers.GetHandler)
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(w, r, baseURL)
	})
	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		handlers.JSONHandler(w, r, baseURL)
	})

	log.Fatal(http.ListenAndServe(host, r))
}
