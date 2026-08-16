// Package handlers implements HTTP handlers for the URL shortener service.
//
// The package provides handlers for all REST API endpoints:
//   - GET  /{id}         — redirect to original URL by short key
//   - POST /            — plain-text URL shortening
//   - POST /api/shorten — JSON URL shortening
//   - POST /api/shorten/batch — batch JSON URL shortening
//   - GET  /ping        — health check
//   - GET  /api/user/urls — retrieve all URLs for the current user
//   - DELETE /api/user/urls — asynchronously delete URLs for the current user
//
// Handlers use object pools (sync.Pool) for Input, Output, and byte buffers
// to minimize garbage collector pressure under high load.
package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	audit "github.com/andrea20024/go-musthave-shortener-tpl/internal/audit"
	auth "github.com/andrea20024/go-musthave-shortener-tpl/internal/auth"
	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
)

// Input represents the request body for the JSON URL shortening endpoint.
type Input struct {
	URL string `json:"url"`
}

// Output represents the response body for the JSON URL shortening endpoint.
type Output struct {
	Result string `json:"result"`
}

// BatchInput represents a single entry in a batch shortening request.
type BatchInput struct {
	// CorrelationID is an arbitrary identifier returned unchanged in the
	// corresponding BatchOutput, allowing the caller to correlate requests
	// and responses.
	CorrelationID string `json:"correlation_id"`
	// OriginalURL is the long URL to be shortened.
	OriginalURL string `json:"original_url"`
}

// BatchOutput represents a single entry in a batch shortening response.
type BatchOutput struct {
	// CorrelationID echoes the value from the corresponding BatchInput.
	CorrelationID string `json:"correlation_id"`
	// ShortURL is the newly created short URL.
	ShortURL string `json:"short_url"`
}

// inputPool is a sync.Pool of reusable Input structs to reduce allocations.
var inputPool = sync.Pool{
	New: func() interface{} { return &Input{} },
}

// outputPool is a sync.Pool of reusable Output structs to reduce allocations.
var outputPool = sync.Pool{
	New: func() interface{} { return &Output{} },
}

// shortURLBytesPool is a sync.Pool of reusable byte buffers used for
// generating random short URL keys.
var shortURLBytesPool = sync.Pool{
	New: func() interface{} { return make([]byte, 6) },
}

// GetHandler handles GET requests to redirect to the original URL by short key.
//
// Expected URL pattern: GET /{id} where {id} is the short URL key.
// On success it returns HTTP 307 Temporary Redirect with the Location header
// set to the original URL. It also emits an audit event with action "follow".
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

// PostHandler handles plain-text POST requests to shorten a URL.
//
// Expected URL pattern: POST / with the original URL as plain-text body.
// Returns HTTP 201 Created with the short URL as plain-text body.
// Returns HTTP 409 Conflict if the original URL already exists.
func PostHandler(w http.ResponseWriter, req *http.Request, config *config.Config, repo storage.Repository, notifier *audit.Notifier) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only POST method", http.StatusBadRequest)
		return
	}

	buf := make([]byte, 1024)
	n, err := req.Body.Read(buf)
	if err != nil && n == 0 {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	url := string(buf[:n])

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

// JSONHandler handles JSON POST requests to shorten a URL.
//
// Expected URL pattern: POST /api/shorten with a JSON body containing
// {"url": "https://example.com"}.
// Returns HTTP 201 Created with a JSON body containing {"result": "https://shortener/abcd12"}.
// Returns HTTP 409 Conflict if the original URL already exists.
func JSONHandler(w http.ResponseWriter, req *http.Request, config *config.Config, repo storage.Repository, notifier *audit.Notifier) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only POST method", http.StatusBadRequest)
		return
	}

	inputBody := inputPool.Get().(*Input)
	defer inputPool.Put(inputBody)
	if err := json.NewDecoder(req.Body).Decode(inputBody); err != nil {
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
				res := outputPool.Get().(*Output)
				res.Result = config.BaseURL + "/" + existingKey
				resp, err := json.Marshal(res)
				outputPool.Put(res)
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

	res := outputPool.Get().(*Output)
	res.Result = config.BaseURL + "/" + shortURL
	resp, err := json.Marshal(res)
	outputPool.Put(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}

// PingHandler handles health-check requests.
//
// Expected URL pattern: GET /ping
// Returns HTTP 200 OK if the storage backend is reachable, HTTP 500 otherwise.
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

// GenerateShortURL generates a cryptographically secure random short URL key
// using 6 bytes from crypto/rand, encoded as base62-friendly URL-safe base64.
func GenerateShortURL() (string, error) {
	bytes := shortURLBytesPool.Get().([]byte)
	defer shortURLBytesPool.Put(bytes)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:8], nil
}

// BatchHandler handles batch URL shortening via JSON POST requests.
//
// Expected URL pattern: POST /api/shorten/batch with a JSON array body:
// [
//
//	{"correlation_id": "abc", "original_url": "https://example.com"},
//	{"correlation_id": "def", "original_url": "https://golang.org"}
//
// ]
// Returns HTTP 201 Created with a JSON array of BatchOutput.
func BatchHandler(w http.ResponseWriter, req *http.Request, config *config.Config, repo storage.Repository) {
	if req.Method != http.MethodPost {
		http.Error(w, "Only POST method", http.StatusBadRequest)
		return
	}

	var batchInputs []BatchInput
	if err := json.NewDecoder(req.Body).Decode(&batchInputs); err != nil {
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
			ShortURL:      config.BaseURL + "/" + shortURL,
		}
		results = append(results, result)
	}

	if err := repo.AddBatch(urls, userID); err != nil {
		if repo.IsDuplicateError(err) {
			http.Error(w, "Duplicate URL in batch", http.StatusConflict)
		} else {
			http.Error(w, "Add batch failed", http.StatusInternalServerError)
		}
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

// GetURLByUserHandler retrieves all URLs shortened by the authenticated user.
//
// Expected URL pattern: GET /api/user/urls
// Returns HTTP 200 OK with a JSON array of UserURL entries, or HTTP 204 No
// Content if the user has no shortened URLs.
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

// getUserID extracts the authenticated user ID from the request context.
// Returns the userID and true if a valid cookie was present, or empty string and false otherwise.
func getUserID(req *http.Request) (string, bool) {
	userID, ok := req.Context().Value(auth.UserIDContextKey).(string)
	return userID, ok
}

// DeleteURLsHandler asynchronously deletes URLs for the authenticated user.
//
// Expected URL pattern: DELETE /api/user/urls with a JSON array of short keys:
// ["abc123", "def456"]
// Returns HTTP 202 Accepted if the delete task was enqueued, or HTTP 400/401
// on validation / auth errors.
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
