package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"runbin/internal/config"
	"runbin/internal/model"
	"runbin/internal/repository"

	"github.com/docker/docker/client"
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

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}

	log.Println("Thread start!")

	for {
		select {
		case <-ticker.C:
			if task, err := w.repo.GetTask(ctx); err == nil {
				if task == nil {
					continue
				}
				if err := w.handleTask(ctx, task, cli); err != nil {
					log.Printf("Worker error at PasteID: %s, error: %v\n", task.ID, err)
				}
				if err := w.repo.Update(task); err != nil {
					log.Printf("Update error at PasteID: %s, error: %v\n", task.ID, err)
				}
			} else {
				log.Printf("Worker get task error: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) handleTask(ctx context.Context, task *model.Paste, cli *client.Client) error {
	log.Printf("Hangling task %s for language %s", task.ID, task.Language)

	task.Status = model.StatusRunning
	task.BackEnd = w.cfg.Name
	w.repo.Update(task)

	var err error

	switch task.Language {
	case "c++20":
		err = w.RunCppTask(ctx, task, cli)
	default:
		err = fmt.Errorf("Unsupported language '%s'", task.Language)
	}

	if err != nil {
		task.Status = model.StatusUnknownError
		task.CompileLog = err.Error()
	}

	log.Printf("Judged task %s, status: %s, runtime: %dms, memory: %dkb", task.ID, task.Status, task.ExecutionTimeMs, task.MemoryUsageKb)

	return err
}
