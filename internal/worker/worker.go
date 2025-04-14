package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"runbin/internal/config"
	"runbin/internal/model"
	"runbin/internal/repository"
)

type Worker struct {
	repo repository.PasteRepository
	cfg  *config.WorkerConfig
}

func NewWorker(repo repository.PasteRepository, cfg *config.WorkerConfig) *Worker {
	return &Worker{
		repo: repo,
		cfg:  cfg,
	}
}

func (w *Worker) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// run n process
	for range w.cfg.Process {
		go w.processTasks(ctx)
	}

	<-ctx.Done()
}

func (w *Worker) processTasks(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("Thread start!")

	for {
		select {
		case <-ticker.C:
			if task, err := w.repo.GetTask(ctx); err == nil {
				if task == nil {
					continue
				}
				if err := w.handleTask(ctx, task); err != nil {
					log.Printf("Worker error at PasteID: %s, error: %v\n", task.ID, err)
				}
				if err := w.repo.Update(task); err != nil {
					log.Printf("Update error at PasteID: %s, error: %v\n", task.ID, err)
				}
				log.Printf("Judged task %s", task.ID)
			} else {
				log.Printf("Worker get task error: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) handleTask(ctx context.Context, task *model.Paste) error {
	log.Printf("Hangling task %s for language %s", task.ID, task.Language)

	task.Status = model.StatusRunning
	task.BackEnd = w.cfg.Name
	w.repo.Update(task)


	var err error

	switch task.Language {
	case "c++20":
		err = w.RunCppTask(ctx, task)
	default:
		err = fmt.Errorf("Unsupported language '%s'", task.Language)
	}

	if err != nil {
		task.Status = model.StatusUnknownError
		task.CompileLog = err.Error()
	}
	return err
}
