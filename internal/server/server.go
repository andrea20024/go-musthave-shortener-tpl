// Package server initializes the HTTP server for the URL shortener service.
//
// It sets up the routing layer using chi, wires up storage backends
// (memory, file-based, PostgreSQL), configures middleware (logging, gzip
// compression, cookie-based authentication), and starts listening for
// incoming HTTP requests.
package server

import (
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"net/http/pprof"

	audit "github.com/andrea20024/go-musthave-shortener-tpl/internal/audit"
	auth "github.com/andrea20024/go-musthave-shortener-tpl/internal/auth"
	compress "github.com/andrea20024/go-musthave-shortener-tpl/internal/compress"
	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	handlers "github.com/andrea20024/go-musthave-shortener-tpl/internal/handler"
	logger "github.com/andrea20024/go-musthave-shortener-tpl/internal/logger"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Start initializes the URL shortener HTTP server and blocks until a
// termination signal (SIGINT / SIGTERM) is received.
//
// The function performs the following initialization steps:
//   - Selects the storage backend (memory → file → PostgreSQL, highest priority wins)
//   - Migrates the PostgreSQL schema if the database backend is used
//   - Loads events from the file storage into the in-memory repository
//   - Configures middleware: logging, gzip compression, cookie authentication
//   - Registers all HTTP routes
//   - Starts a background worker for asynchronous URL deletion
//   - Sets up audit log receivers (file and/or HTTP)
//   - Optionally serves pprof profiling endpoints under /debug/pprof/
//
// Routes:
//
//	GET    /{id}                — redirect to the original URL by short key
//	POST   /                    — plain-text URL shortening
//	POST   /api/shorten         — JSON URL shortening
//	POST   /api/shorten/batch   — batch JSON URL shortening
//	GET    /ping                — health check
//	GET    /api/user/urls       — list all URLs for the current user
//	DELETE /api/user/urls       — asynchronously delete URLs for the current user
//	GET    /debug/pprof/*       — Go profiler endpoints (when enabled)
func Start(config *config.Config) {
	r := chi.NewRouter()

	auth.Init(config.AuthSecret)

	var repo storage.Repository

	repo = storage.NewMapRepository()

	if config.FilePath != "" {
		newRepo, err := storage.NewFileRepository(config.FilePath)
		if err == nil {
			repo = newRepo
			config.StoreType = "file"
		}
	}

	if config.DB != "" {
		newRepo := storage.Init(config.DB)
		if newRepo != nil {
			repo = newRepo
			config.StoreType = "db"
		}
	}

	logg, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	logger.InitLogger(logg)

	sugar := *logg.Sugar()
	sugar.Infow("Starting server", "addr", config.Host)

	if config.FilePath != "" && config.StoreType != "file" {
		consumer, err := storage.NewConsumer(config.FilePath)
		if err != nil {
			sugar.Errorf("NewConsumer error: %v", err)
		} else {
			for {
				event, err := consumer.ReadEvent()
				if err != nil {
					if err == io.EOF {
						break
					}
					continue
				}
				if event == nil {
					break
				}
				repo.Add(event.ShortURL, event.OriginalURL, "")
			}
		}
	}

	worker := handlers.NewWorker(config.WorkerBufferSize, repo)

	notifier := audit.NewNotifier()
	notifier.Start()
	if config.AuditFile != "" {
		fileReceiver, err := audit.NewFileReceiver(config.AuditFile)
		if err == nil {
			notifier.Attach(fileReceiver)
		}
	}
	if config.AuditURL != "" {
		httpReceiver := audit.NewHTTPReceiver(config.AuditURL)
		notifier.Attach(httpReceiver)
	}

	r.Use(logger.WithLogging)
	r.Use(compress.GzipHandle)
	r.Use(auth.CookieMiddleware)

	r.Get("/{id}", func(w http.ResponseWriter, req *http.Request) {
		handlers.GetHandler(w, req, repo, notifier)
	})
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(w, r, config, repo, notifier)
	})
	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		handlers.JSONHandler(w, r, config, repo, notifier)
	})
	r.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
		handlers.PingHandler(w, req, repo)
	})
	r.Post("/api/shorten/batch", func(w http.ResponseWriter, r *http.Request) {
		handlers.BatchHandler(w, r, config, repo)
	})
	r.Get("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		handlers.GetURLByUserHandler(w, r, config, repo)
	})
	r.Delete("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		handlers.DeleteURLsHandler(w, r, config, repo, worker)
	})

	// profiler
	r.HandleFunc("/debug/pprof/heap", func(w http.ResponseWriter, r *http.Request) {
		runtime.GC()
		pprof.Profile(w, r)
	})
	r.HandleFunc("/debug/pprof/allocs", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("allocs").ServeHTTP(w, r)
	})
	r.HandleFunc("/debug/pprof/gc", func(w http.ResponseWriter, r *http.Request) {
		pprof.Index(w, r)
	})

	go func() {
		if err := http.ListenAndServe(config.Host, r); err != nil {
			sugar.Fatalf("Server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	worker.Shutdown()
	notifier.Stop()
	sugar.Info("Server stopped")
}
