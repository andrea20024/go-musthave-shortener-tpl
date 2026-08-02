package handlers

import (
	"sync"

	"github.com/andrea20024/go-musthave-shortener-tpl/internal/logger"
	storage "github.com/andrea20024/go-musthave-shortener-tpl/internal/repository"
)

type DeleteTask struct {
	userID string
	keys   []string
}

type Worker struct {
	tasks chan DeleteTask
	repo  storage.Repository
	wg    sync.WaitGroup
}

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

func (w *Worker) submit(task DeleteTask) bool {
	select {
	case w.tasks <- task:
		return true
	default:
		return false
	}
}

func (w *Worker) Shutdown() {
	close(w.tasks)
	w.wg.Wait()
}
