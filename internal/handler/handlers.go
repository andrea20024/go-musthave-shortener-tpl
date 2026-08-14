package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	auth "github.com/andrea20024/go-musthave-shortener-tpl/internal/auth"
	audit "github.com/andrea20024/go-musthave-shortener-tpl/internal/audit"
	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
)

type Input struct {
	URL string `json:"url"`
}

type Output struct {
	Result string `json:"result"`
}

type BatchInput struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchOutput struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func GetHandler(w http.ResponseWriter, req *http.Request, repo storage.Repository, notifier *audit.Notifier) {
	if req.Method != http.MethodGet {
		http.Error(w, "Only GET method", http.StatusBadRequest)
		return
	}

	shortURL := req.URL.Path[1:]

	url, err := repo.Get(shortURL)
	if err != nil {
		if repo.IsDeletedError(err) {
			w.WriteHeader(http.StatusGone)
			return
		}
		http.Error(w, "Url not found!", http.StatusBadRequest)
		return
	}

	userID, _ := getUserID(req)
	if notifier != nil {
		notifier.NotifyAll(audit.Event{
			Timestamp: time.Now().Unix(),
			Action:    "follow",
			UserID:    userID,
			URL:       url,
		})
	}

	w.Header().Add("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func PostHandler(w http.ResponseWriter, req *http.Request, config *config.Config, repo storage.Repository, notifier *audit.Notifier) {
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

	userID, _ := getUserID(req)

	err = repo.Add(shortURL, url, userID)
	if err != nil {
		if repo.IsDuplicateError(err) {
			existingKey, err := repo.GetKeyByURL(url)
			if err == nil && existingKey != "" {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(config.BaseURL + "/" + existingKey))
				return
			}
		}
		http.Error(w, "Add url failed", http.StatusInternalServerError)
		return
	}

	if notifier != nil {
		notifier.NotifyAll(audit.Event{
			Timestamp: time.Now().Unix(),
			Action:    "shorten",
			UserID:    userID,
			URL:       url,
		})
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Authorization", userID)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(config.BaseURL + "/" + shortURL))
}

func JSONHandler(w http.ResponseWriter, req *http.Request, config *config.Config, repo storage.Repository, notifier *audit.Notifier) {
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

	userID, _ := getUserID(req)

	err = repo.Add(shortURL, url, userID)
	if err != nil {
		if repo.IsDuplicateError(err) {
			existingKey, err := repo.GetKeyByURL(url)
			if err == nil && existingKey != "" {
				res := Output{Result: fmt.Sprintf("%s/%s", config.BaseURL, existingKey)}
				resp, err := json.Marshal(res)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				w.Write(resp)
				return
			}
		}
		http.Error(w, "Add url failed", http.StatusInternalServerError)
		return
	}

	if notifier != nil {
		notifier.NotifyAll(audit.Event{
			Timestamp: time.Now().Unix(),
			Action:    "shorten",
			UserID:    userID,
			URL:       url,
		})
	}

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

func PingHandler(w http.ResponseWriter, req *http.Request, repo storage.Repository) {
	if req.Method != http.MethodGet {
		http.Error(w, "Only GET method", http.StatusBadRequest)
		return
	}

	err := repo.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func GenerateShortURL() (string, error) {
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}

func BatchHandler(w http.ResponseWriter, req *http.Request, config *config.Config, repo storage.Repository) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only POST method", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	var batchInputs []BatchInput
	if err = json.Unmarshal(body, &batchInputs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(batchInputs) == 0 {
		http.Error(w, "Empty batch", http.StatusBadRequest)
		return
	}

	results := make([]BatchOutput, 0, len(batchInputs))
	urls := make(map[string]string)
	userID, _ := getUserID(req)
	for _, input := range batchInputs {
		shortURL, err := GenerateShortURL()
		if err != nil {
			http.Error(w, "Generate url failed", http.StatusInternalServerError)
			return
		}

		urls[shortURL] = input.OriginalURL

		result := BatchOutput{
			CorrelationID: input.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", config.BaseURL, shortURL),
		}
		results = append(results, result)
	}

	err = repo.AddBatch(urls, userID)
	if err != nil {
		if repo.IsDuplicateError(err) {
			http.Error(w, "Duplicate URL in batch", http.StatusConflict)
			return
		}
		http.Error(w, "Add batch failed", http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(results)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}

func GetURLByUserHandler(w http.ResponseWriter, req *http.Request, config *config.Config, repo storage.Repository) {
	if req.Method != http.MethodGet {
		http.Error(w, "Only GET method", http.StatusBadRequest)
		return
	}

	userID, ok := getUserID(req)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	urls, err := repo.GetUserURLs(userID)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	for i := range urls {
		if !strings.Contains(urls[i].ShortURL, "://") {
			urls[i].ShortURL = config.BaseURL + "/" + urls[i].ShortURL
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(urls)
}

func getUserID(req *http.Request) (string, bool) {
	userID, ok := req.Context().Value(auth.UserIDContextKey).(string)
	return userID, ok
}

func DeleteURLsHandler(w http.ResponseWriter, req *http.Request, config *config.Config, repo storage.Repository, worker *Worker) {
	if req.Method != http.MethodDelete {
		http.Error(w, "Only DELETE method", http.StatusBadRequest)
		return
	}

	userID, ok := getUserID(req)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var keys []string
	if err := json.NewDecoder(req.Body).Decode(&keys); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(keys) == 0 {
		http.Error(w, "Empty array", http.StatusBadRequest)
		return
	}

	if worker == nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	sent := worker.submit(DeleteTask{
		userID: userID,
		keys:   keys,
	})
	if !sent {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
