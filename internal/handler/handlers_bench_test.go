package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	audit "github.com/andrea20024/go-musthave-shortener-tpl/internal/audit"
	auth "github.com/andrea20024/go-musthave-shortener-tpl/internal/auth"
	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
)

// ============================================================================
// Benchmarks для auth.GenerateShortURL
// ============================================================================

func BenchmarkGenerateShortURL(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = auth.GenerateShortURL()
	}
}

// ============================================================================
// Benchmarks для JSONHandler
// ============================================================================

func BenchmarkJSONHandler(b *testing.B) {
	cfg := config.InitConfig()
	cfg.BaseURL = "http://localhost:8080"
	repo := storage.NewMapRepository()
	notifier := audit.NewNotifier()
	body := []byte(`{"url":"https://example.com/test"}`)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		JSONHandler(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body)),
			cfg, repo, notifier,
		)
	}
}

func BenchmarkJSONHandlerNoNotifier(b *testing.B) {
	cfg := config.InitConfig()
	cfg.BaseURL = "http://localhost:8080"
	repo := storage.NewMapRepository()
	body := []byte(`{"url":"https://example.com/test"}`)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		JSONHandler(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body)),
			cfg, repo, nil,
		)
	}
}

// ============================================================================
// Benchmarks для BatchHandler
// ============================================================================

func BenchmarkBatchHandler(b *testing.B) {
	cfg := config.InitConfig()
	cfg.BaseURL = "http://localhost:8080"
	repo := storage.NewMapRepository()

	batchData := []BatchInput{
		{CorrelationID: "corr-1", OriginalURL: "https://example.com/1"},
		{CorrelationID: "corr-2", OriginalURL: "https://example.com/2"},
		{CorrelationID: "corr-3", OriginalURL: "https://example.com/3"},
		{CorrelationID: "corr-4", OriginalURL: "https://example.com/4"},
		{CorrelationID: "corr-5", OriginalURL: "https://example.com/5"},
	}
	body, _ := json.Marshal(batchData)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		BatchHandler(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body)),
			cfg, repo,
		)
	}
}

func BenchmarkBatchHandlerLarge(b *testing.B) {
	cfg := config.InitConfig()
	cfg.BaseURL = "http://localhost:8080"
	repo := storage.NewMapRepository()

	batchData := make([]BatchInput, 50)
	for i := range batchData {
		batchData[i] = BatchInput{
			CorrelationID: string(rune('a' + i%26)),
			OriginalURL:   "https://example.com/batch-large-" + string(rune('a'+i%26)),
		}
	}
	body, _ := json.Marshal(batchData)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		BatchHandler(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body)),
			cfg, repo,
		)
	}
}

// ============================================================================
// Benchmarks для GetUserURLs
// ============================================================================

func BenchmarkGetUserURLs(b *testing.B) {
	repo := storage.NewMapRepository()
	userID := "test-user-123"

	// Наполняем репозиторий до бенчмарка
	for i := 0; i < 100; i++ {
		shortURL, _ := auth.GenerateShortURL()
		repo.Add(shortURL, "https://example.com/user/"+string(rune('a'+i%26)), userID)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = repo.GetUserURLs(userID)
	}
}

// ============================================================================
// Benchmarks для GetHandler (редирект)
// ============================================================================

func BenchmarkGetHandler(b *testing.B) {
	repo := storage.NewMapRepository()
	notifier := audit.NewNotifier()

	shortURL, _ := auth.GenerateShortURL()
	repo.Add(shortURL, "https://example.com/target", "user-1")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		GetHandler(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, "/"+shortURL, nil),
			repo, notifier,
		)
	}
}

// ============================================================================
// Benchmarks для FileRepository (disk-backed operations)
// ============================================================================

func BenchmarkMapRepository_Add(b *testing.B) {
	repo := storage.NewMapRepository()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		shortURL, _ := auth.GenerateShortURL()
		repo.Add(shortURL, "https://example.com/add-test", "user-add")
	}
}

// ============================================================================
// Benchmarks для конкурентного доступа (MapRepository)
// ============================================================================

func BenchmarkMapRepository_ConcurrentAdd(b *testing.B) {
	repo := storage.NewMapRepository()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			shortURL, _ := auth.GenerateShortURL()
			repo.Add(shortURL, "https://example.com/concurrent", "user-concurrent")
		}
	})
}

// ============================================================================
// Benchmarks для алгоритма IsDuplicateError
// ============================================================================

func BenchmarkIsDuplicateError(b *testing.B) {
	repo := storage.NewMapRepository()
	dupErr := &storage.DuplicateError{}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if !repo.IsDuplicateError(dupErr) {
			b.Fatal("expected true")
		}
	}
}

// ============================================================================
// Benchmarks для PostHandler (plain text)
// ============================================================================

func BenchmarkPostHandler(b *testing.B) {
	cfg := config.InitConfig()
	cfg.BaseURL = "http://localhost:8080"
	repo := storage.NewMapRepository()
	body := []byte("https://example.com/post-test")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		PostHandler(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)),
			cfg, repo, nil,
		)
	}
}
