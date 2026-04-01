package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
)

type Input struct {
	URL string `json:"url"`
}

type Output struct {
	Result string `json:"result"`
}

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

func PostHandler(w http.ResponseWriter, req *http.Request, config *config.Config) {
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

	producer, err := storage.NewProducer(config.FilePath)
	if err != nil {
		http.Error(w, "file error", http.StatusInternalServerError)
		return
	}
	event := GetEnevt(shortURL, url)
	err = producer.WriteEvent(&event)
	if err != nil {
		http.Error(w, "file error", http.StatusInternalServerError)
		return
	}
	defer producer.Close()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(config.BaseURL + "/" + shortURL))
}

func JSONHandler(w http.ResponseWriter, req *http.Request, config *config.Config) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only POST method", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	var inputBody Input
	if err = json.Unmarshal(body, &inputBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	url := inputBody.URL
	shortURL, err := GenerateShortURL()
	if err != nil {
		http.Error(w, "Generate url failed", http.StatusInternalServerError)
		return
	}

	storage.Add(shortURL, url)

	producer, err := storage.NewProducer(config.FilePath)
	if err != nil {
		http.Error(w, "file error", http.StatusInternalServerError)
		return
	}
	event := GetEnevt(shortURL, url)
	err = producer.WriteEvent(&event)
	if err != nil {
		http.Error(w, "file error", http.StatusInternalServerError)
		return
	}
	defer producer.Close()

	res := Output{Result: fmt.Sprintf("%s/%s", config.BaseURL, shortURL)}
	resp, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}

func GenerateShortURL() (string, error) {
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}

func GetEnevt(shortURL string, url string) storage.Event {
	currentTime := time.Now()
	intFromTime := currentTime.Unix()
	return storage.Event{
		UUID:        strconv.Itoa(int(intFromTime)),
		ShortURL:    shortURL,
		OriginalURL: url,
	}
}
