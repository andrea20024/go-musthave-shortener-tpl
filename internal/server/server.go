package server

import (
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	compress "github.com/andrea20024/go-musthave-shortener-tpl/internal/compress"
	config "github.com/andrea20024/go-musthave-shortener-tpl/internal/config"
	handlers "github.com/andrea20024/go-musthave-shortener-tpl/internal/handler"
	logger "github.com/andrea20024/go-musthave-shortener-tpl/internal/logger"
	auth "github.com/andrea20024/go-musthave-shortener-tpl/internal/auth"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

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

	r.Use(logger.WithLogging)
	r.Use(compress.GzipHandle)
	r.Use(auth.CookieMiddleware)

	r.Get("/{id}", func(w http.ResponseWriter, req *http.Request) {
		handlers.GetHandler(w, req, repo)
	})
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(w, r, config, repo)
	})
	r.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		handlers.JSONHandler(w, r, config, repo)
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

	go func() {
		if err := http.ListenAndServe(config.Host, r); err != nil {
			sugar.Fatalf("Server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	worker.Shutdown()
	sugar.Info("Server stopped")
}
