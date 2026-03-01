package main

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
)

var dict = make(map[string]string)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/{id}`, getHandler)
	mux.HandleFunc(`/`, postHandler)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
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
