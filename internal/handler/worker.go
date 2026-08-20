package handlers

import (
	"sync"

	"github.com/andrea20024/go-musthave-shortener-tpl/internal/logger"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
)

// DeleteTask represents an asynchronous request to delete multiple URLs
// owned by a specific user.
// generate:reset
type DeleteTask struct {
	userID string
	keys   []string
}

// Worker manages a pool of goroutines that process asynchronous delete tasks.
//
// Tasks are submitted via a buffered channel. If the channel is full, submit
// returns false immediately, allowing the caller to return 503 Service
// Unavailable. Shutdown blocks until all pending tasks are processed.
type Worker struct {
	tasks chan DeleteTask
	repo  storage.Repository
	wg    sync.WaitGroup
}

// NewWorker creates a new Worker with the given channel capacity and storage
// repository. It starts a background goroutine that processes delete tasks.
func NewWorker(capacity int, repo storage.Repository) *Worker {
	w := &Worker{
		tasks: make(chan DeleteTask, capacity),
		repo:  repo,
	}
	go func() {
		for task := range w.tasks {
			w.wg.Add(1)
			if err := w.repo.DeleteUserURLs(task.userID, task.keys); err != nil {
				logger.Sugar().Errorw("delete failed", "userID", task.userID, "error", err)
			}
			w.wg.Done()
		}
	}()
	return w
}

// submit attempts to enqueue a DeleteTask. Returns true if the task was
// accepted, false if the worker's task queue is full.
func (w *Worker) submit(task DeleteTask) bool {
	select {
	case w.tasks <- task:
		return true
	default:
		return false
	}
}

// Shutdown closes the task channel and blocks until all in-flight tasks
// have been processed. This should be called during graceful shutdown.
func (w *Worker) Shutdown() {
	close(w.tasks)
	w.wg.Wait()
}
