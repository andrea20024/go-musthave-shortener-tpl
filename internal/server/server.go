package server

import (
	"log"
	"net/http"

	handlers "github.com/andrea20024/go-musthave-shortener-tpl/internal/handler"
	"github.com/go-chi/chi/v5"
)

func Start(host string, baseURL string) {
	r := chi.NewRouter()
	r.Get("/{id}", handlers.GetHandler)
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(w, r, baseURL)
	})

	log.Fatal(http.ListenAndServe(host, r))
}
