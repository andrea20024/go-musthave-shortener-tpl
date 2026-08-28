package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	auth "github.com/andrea20024/go-musthave-shortener-tpl/internal/auth"
	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
	"github.com/stretchr/testify/assert"
)

func Test_GetHandler(t *testing.T) {
	type params struct {
		code        int
		contentType string
	}
	tests := []struct {
		name   string
		ref    string
		params params
	}{
		{
			name: "test get",
			ref:  "https://practicum.yandex.ru",
			params: params{
				code:        http.StatusTemporaryRedirect,
				contentType: "text/plain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortURL, err := GenerateShortURL()
			if err != nil {
				return
			}

			repo := storage.NewMapRepository()
			repo.Add(shortURL, tt.ref, "test-user-id")
			req := httptest.NewRequest(http.MethodGet, "/"+shortURL, nil)
			w := httptest.NewRecorder()

			GetHandler(w, req, repo, nil)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.params.code, res.StatusCode)
			assert.Equal(t, tt.ref, res.Header.Get("location"))
		})
	}
}

func TestPostHandler(t *testing.T) {
	type params struct {
		code        int
		contentType string
	}
	tests := []struct {
		name   string
		body   string
		config config.Config
		params params
	}{
		{
			name:   "test post",
			body:   "https://practicum.yandex.ru",
			config: *config.InitConfig(),
			params: params{
				code:        http.StatusCreated,
				contentType: "text/plain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			repo := storage.NewMapRepository()
			PostHandler(w, req, &tt.config, repo, nil)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.params.code, res.StatusCode)
			assert.Equal(t, tt.params.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestPingHandler(t *testing.T) {
	repo := storage.NewMapRepository()

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "valid GET request",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid POST request",
			method:     http.MethodPost,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/ping", nil)
			w := httptest.NewRecorder()

			PingHandler(w, req, repo)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}

func TestPingHandler_RepoError(t *testing.T) {
	mockRepo := &mockRepoForPing{}

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	PingHandler(w, req, mockRepo)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}

func TestGetURLByUserHandler(t *testing.T) {
	cfg := config.Config{
		BaseURL: "https://shortener.local",
	}

	repo := storage.NewMapRepository()

	short1, _ := GenerateShortURL()
	short2, _ := GenerateShortURL()
	repo.Add(short1, "https://example.com/1", "user123")
	repo.Add(short2, "https://example.com/2", "user123")

	tests := []struct {
		name       string
		method     string
		userID     string
		wantStatus int
		wantCount  int
	}{
		{
			name:       "valid GET request with user URLs",
			method:     http.MethodGet,
			userID:     "user123",
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "valid GET request with empty user URLs",
			method:     http.MethodGet,
			userID:     "unknown-user",
			wantStatus: http.StatusNoContent,
			wantCount:  0,
		},
		{
			name:       "invalid POST request",
			method:     http.MethodPost,
			userID:     "user123",
			wantStatus: http.StatusBadRequest,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/user/urls", nil)
			req = req.WithContext(context.WithValue(req.Context(), auth.UserIDContextKey, tt.userID))
			w := httptest.NewRecorder()

			GetURLByUserHandler(w, req, &cfg, repo)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)

			if tt.wantStatus == http.StatusOK {
				var urls []storage.UserURL
				err := json.NewDecoder(res.Body).Decode(&urls)
				assert.NoError(t, err)
				assert.Len(t, urls, tt.wantCount)

				for _, u := range urls {
					assert.Contains(t, u.ShortURL, "https://shortener.local")
				}
			}
		})
	}
}

func TestDeleteURLsHandler(t *testing.T) {
	cfg := config.Config{
		BaseURL: "https://shortener.local",
	}

	repo := storage.NewMapRepository()
	worker := NewWorker(10, repo)
	defer worker.Shutdown()

	short1, _ := GenerateShortURL()
	repo.Add(short1, "https://example.com/1", "user123")

	tests := []struct {
		name       string
		method     string
		userID     string
		body       string
		wantStatus int
	}{
		{
			name:       "valid DELETE request",
			method:     http.MethodDelete,
			userID:     "user123",
			body:       `["` + short1 + `"]`,
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "invalid POST request",
			method:     http.MethodPost,
			userID:     "user123",
			body:       `[]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty array",
			method:     http.MethodDelete,
			userID:     "user123",
			body:       `[]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			method:     http.MethodDelete,
			userID:     "user123",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthorized user",
			method:     http.MethodDelete,
			userID:     "",
			body:       `["key1"]`,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/user/urls", bytes.NewBufferString(tt.body))
			if tt.userID != "" {
				req = req.WithContext(context.WithValue(req.Context(), auth.UserIDContextKey, tt.userID))
			}
			w := httptest.NewRecorder()

			DeleteURLsHandler(w, req, &cfg, repo, worker)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}

func TestDeleteURLsHandler_NoWorker(t *testing.T) {
	cfg := config.Config{
		BaseURL: "https://shortener.local",
	}

	repo := storage.NewMapRepository()

	short1, _ := GenerateShortURL()
	repo.Add(short1, "https://example.com/1", "user123")

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString(`["`+short1+`"]`))
	req = req.WithContext(context.WithValue(req.Context(), auth.UserIDContextKey, "user123"))
	w := httptest.NewRecorder()

	DeleteURLsHandler(w, req, &cfg, repo, nil)
	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
}

func TestWorker_NewWorker(t *testing.T) {
	repo := storage.NewMapRepository()
	worker := NewWorker(10, repo)
	defer worker.Shutdown()

	if worker == nil {
		t.Fatal("NewWorker() returned nil")
	}
	if worker.tasks == nil {
		t.Error("Worker.tasks is nil")
	}
}

func TestWorker_SubmitAndShutdown(t *testing.T) {
	repo := storage.NewMapRepository()

	key1, _ := GenerateShortURL()
	key2, _ := GenerateShortURL()
	repo.Add(key1, "https://example.com/1", "user1")
	repo.Add(key2, "https://example.com/2", "user1")

	worker := NewWorker(10, repo)

	task := DeleteTask{
		userID: "user1",
		keys:   []string{key1, key2},
	}

	sent := worker.submit(task)
	if !sent {
		t.Error("worker.submit() returned false for available worker")
	}

	time.Sleep(50 * time.Millisecond)
	worker.Shutdown()

	_, err := repo.Get(key1)
	if !repo.IsDeletedError(err) {
		t.Error("URL key1 should be marked as deleted")
	}

	_, err = repo.Get(key2)
	if !repo.IsDeletedError(err) {
		t.Error("URL key2 should be marked as deleted")
	}
}

func TestWorker_Submit_FullQueue(t *testing.T) {
	repo := storage.NewMapRepository()
	worker := NewWorker(1, repo)

	task1 := DeleteTask{userID: "user1", keys: []string{"key1"}}
	sent1 := worker.submit(task1)
	if !sent1 {
		t.Error("First submit should succeed")
	}

	task2 := DeleteTask{userID: "user2", keys: []string{"key2"}}
	sent2 := worker.submit(task2)
	if sent2 {
		t.Error("Second submit should fail when queue is full")
	}

	worker.Shutdown()
}

func TestGenerateShortURL(t *testing.T) {
	url1, err := GenerateShortURL()
	assert.NoError(t, err)
	assert.Len(t, url1, 8)

	url2, err := GenerateShortURL()
	assert.NoError(t, err)
	assert.Len(t, url2, 8)

	assert.NotEqual(t, url1, url2)
}

// mockRepoForPing is a mock repository that returns error on Ping()
type mockRepoForPing struct{}

func (m *mockRepoForPing) Add(key, url, userID string) error {
	return nil
}

func (m *mockRepoForPing) AddBatch(urls map[string]string, userID string) error {
	return nil
}

func (m *mockRepoForPing) Get(key string) (string, error) {
	return "", nil
}

func (m *mockRepoForPing) GetKeyByURL(url string) (string, error) {
	return "", nil
}

func (m *mockRepoForPing) GetUserURLs(userID string) ([]storage.UserURL, error) {
	return nil, nil
}

func (m *mockRepoForPing) DeleteUserURLs(userID string, keys []string) error {
	return nil
}

func (m *mockRepoForPing) Ping() error {
	return assert.AnError
}

func (m *mockRepoForPing) IsDuplicateError(err error) bool {
	return false
}

func (m *mockRepoForPing) IsDeletedError(err error) bool {
	return false
}

func (m *mockRepoForPing) Shutdown() error {
	return nil
}

func (m *mockRepoForPing) Stats() (int, int, error) {
	return 0, 0, nil
}
