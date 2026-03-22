package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"

	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
)

func GetHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Only GET method", http.StatusBadRequest)
		return
	}

	shortURL := req.URL.Path[1:]

	var url = storage.Get(shortURL)
	if url != "" {
		w.Header().Add("Location", url)
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	} else {
		http.Error(w, "Url not found!", http.StatusBadRequest)
	}
}

func PostHandler(w http.ResponseWriter, req *http.Request, BaseURL string) {
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
	shortURL, err := GenerateShortURL()
	if err != nil {
		http.Error(w, "Generate url failed", http.StatusInternalServerError)
		return
	}

	storage.Add(shortURL, url)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(BaseURL + "/" + shortURL))
}

func GenerateShortURL() (string, error) {
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}
