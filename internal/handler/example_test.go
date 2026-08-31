package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"

	audit "github.com/andrea20024/go-musthave-shortener-tpl/internal/audit"
	auth "github.com/andrea20024/go-musthave-shortener-tpl/internal/auth"
	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
)

// ExampleJSONHandler demonstrates how to use the JSONHandler for URL shortening.
// It shows a complete flow: creating a request with a JSON body, sending it to
// the handler, and receiving the shortened URL in the response.
//
// This example corresponds to the POST /api/shorten endpoint.
func ExampleJSONHandler() {
	// Initialize configuration and repository
	cfg := config.InitConfig()
	repo := storage.NewMapRepository()
	notifier := audit.NewNotifier()

	// Create a JSON request body with the URL to shorten
	jsonBody := `{"url":"https://yandex.ru"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call the JSONHandler
	JSONHandler(w, req, cfg, repo, notifier)

	// Get the response
	res := w.Result()
	defer res.Body.Close()

	// Read and print the response body
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	fmt.Println("Status:", res.StatusCode)
	fmt.Println("Body:", buf.String())
}

// ExampleBatchHandler demonstrates how to use the BatchHandler for bulk URL
// shortening. It shows a complete flow: sending a batch of URLs with correlation
// IDs, and receiving the shortened URLs back.
//
// This example corresponds to the POST /api/shorten/batch endpoint.
func ExampleBatchHandler() {
	// Initialize configuration and repository
	cfg := config.InitConfig()
	repo := storage.NewMapRepository()

	// Create a JSON request body with multiple URLs to shorten
	jsonBody := `[
		{"correlation_id":"abc","original_url":"https://yandex.ru"},
		{"correlation_id":"def","original_url":"https://golang.org"}
	]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call the BatchHandler
	BatchHandler(w, req, cfg, repo)

	// Get the response
	res := w.Result()
	defer res.Body.Close()

	// Read and print the response body
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	fmt.Println("Status:", res.StatusCode)
	fmt.Println("Body length:", buf.Len())
}

// ExampleGetHandler demonstrates how to use the GetHandler for URL redirection.
// It shows how a client follows the redirect to get the original URL.
//
// This example corresponds to the GET /{id} endpoint.
func ExampleGetHandler() {
	// Initialize repository and add a test URL
	repo := storage.NewMapRepository()
	originalURL := "https://yandex.ru"

	// Generate a short URL key
	shortKey, _ := auth.GenerateShortURL()
	repo.Add(shortKey, originalURL, "test-user")

	// Create a GET request to the short URL
	req := httptest.NewRequest(http.MethodGet, "/"+shortKey, nil)
	w := httptest.NewRecorder()

	// Call the GetHandler
	GetHandler(w, req, repo, nil)

	// Get the response
	res := w.Result()
	defer res.Body.Close()

	// Print the redirect status and location
	fmt.Println("Status:", res.StatusCode)
	fmt.Println("Location:", res.Header.Get("Location"))
}
