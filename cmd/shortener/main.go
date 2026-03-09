package main

import (
	"flag"

	"github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	"github.com/andrea20024/go-musthave-shortener-tpl/internal/server"
)

func main() {
	config := config.InitConfig()

	flag.StringVar(&config.Host, "a", config.Host, "host")
	flag.StringVar(&config.BaseURL, "b", config.BaseURL, "base url")
	flag.Parse()

	server.Start(config.Host, config.BaseURL)
}

/*
package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"io"
	"log"
	"net/http"

	"github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	"github.com/go-chi/chi/v5"
)

var dict = make(map[string]string)

func main() {

	config := config.InitConfig()

	flag.StringVar(&config.Host, "a", config.Host, "host")
	flag.StringVar(&config.BaseURL, "b", config.BaseURL, "base url")
	flag.Parse()

	r := chi.NewRouter()
	r.Get("/{id}", getHandler)
	r.Post("/", postHandler)

	log.Fatal(http.ListenAndServe(config.Host, r))
}

func getHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Only GET method", http.StatusBadRequest)
		return
	}

	shortURL := req.URL.Path[1:]

	if url, ok := dict[shortURL]; ok {
		w.Header().Add("Location", url)
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	} else {
		http.Error(w, "Url not found!", http.StatusBadRequest)
	}
}

func postHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only POST method", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	url := string(body)
	shortURL, err := generateShortURL()
	if err != nil {
		http.Error(w, "Generate url failed", http.StatusInternalServerError)
		return
	}

	dict[shortURL] = url

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("http://" + req.Host + "/" + shortURL))
}

func generateShortURL() (string, error) {
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}
*/
