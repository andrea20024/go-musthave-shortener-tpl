// Package server initializes the HTTP server for the URL shortener service.
//
// It sets up the routing layer using chi, wires up storage backends
// (memory, file-based, PostgreSQL), configures middleware (logging, gzip
// compression, cookie-based authentication), and starts listening for
// incoming HTTP requests.
package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"net/http/pprof"

	audit "github.com/andrea20024/go-musthave-shortener-tpl/internal/audit"
	auth "github.com/andrea20024/go-musthave-shortener-tpl/internal/auth"
	compress "github.com/andrea20024/go-musthave-shortener-tpl/internal/compress"
	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	handlers "github.com/andrea20024/go-musthave-shortener-tpl/internal/handler"
	logger "github.com/andrea20024/go-musthave-shortener-tpl/internal/logger"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
	svc "github.com/andrea20024/go-musthave-shortener-tpl/internal/service"

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
//	POST   /api/shorten/batch   — batch JSON shortening
//	GET    /ping                — health check
//	GET    /api/user/urls       — list all URLs for the current user
//	DELETE /api/user/urls       — asynchronously delete URLs for the current user
//	GET    /debug/pprof/*       — Go profiler endpoints (when enabled)
//
// HTTPS Support:
//
//	When config.EnableHTTPS is true, the server starts via http.ListenAndServeTLS()
//	using certificates from config.TLSCertFile and config.TLSKeyFile.
//	EnableHTTPS can be set via command-line flag "-s" or environment variable "ENABLE_HTTPS".
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

	srv := &http.Server{
		Addr:         config.Host,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		var err error
		if config.EnableHTTPS {
			if config.TLSCertFile == "" || config.TLSKeyFile == "" {
				sugar.Info("TLS certificates not specified, generating self-signed certificates...")
				if err := svc.GenerateSelfSignedCert("server.crt", "server.key"); err != nil {
					sugar.Fatalf("Failed to generate TLS certificate: %v", err)
				}
				config.TLSCertFile = "server.crt"
				config.TLSKeyFile = "server.key"
			}

			// Verify certificate files exist
			if _, err := os.Stat(config.TLSCertFile); os.IsNotExist(err) {
				sugar.Fatalf("TLS certificate file not found: %s", config.TLSCertFile)
			}
			if _, err := os.Stat(config.TLSKeyFile); os.IsNotExist(err) {
				sugar.Fatalf("TLS key file not found: %s", config.TLSKeyFile)
			}

			sugar.Infow("Starting HTTPS server", "addr", config.Host)
			err = srv.ListenAndServeTLS(config.TLSCertFile, config.TLSKeyFile)
		} else {
			sugar.Infow("Starting HTTP server", "addr", config.Host)
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			sugar.Fatalf("Server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sig

	sugar.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		sugar.Errorf("Server forced to shutdown: %v", err)
	}

	worker.Shutdown()
	notifier.Stop()

	if err := repo.Shutdown(); err != nil {
		sugar.Errorf("Storage shutdown error: %v", err)
	}

	sugar.Info("Server stopped")
}
