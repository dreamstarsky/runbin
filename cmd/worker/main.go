package main

import (
	"log"
	"runbin/internal/config"
	"runbin/internal/repository"
	"runbin/internal/worker"
)

func main() {
	cfg := config.LoadWorker("config/worker.yaml")

	// Initialize storage
	var store repository.PasteRepository
	switch cfg.Storage.Type {
	case "memory":
		log.Fatal("Worker can't use memory repository at now!")
	case "database":
		dbStore, err := repository.NewPostgresStore(cfg.Storage.Database.DSN)
		defer dbStore.Close()
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		store = dbStore
	default:
		log.Fatalf("Unsupported storage type: %s", cfg.Storage.Type)
	}

	work := worker.NewWorker(store, cfg)
	work.Run()
}
