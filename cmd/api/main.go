package main

import (
	"log"
	"strconv"

	"runbin/internal/config"
	"runbin/internal/controller"
	"runbin/internal/repository"
	"runbin/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadApi("config/server.yaml")

	// Initialize storage
	var store repository.PasteRepository
	switch cfg.Storage.Type {
	case "memory":
		store = repository.NewMemoryPasteStore()
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

	pasteHandler := controller.NewPasteHandler(store)

	// Create router engine
	engine := gin.Default()
	engine.SetTrustedProxies(nil)

	// Setup routes
	router.SetupRoutes(engine, pasteHandler)

	// Configure Gin mode based on environment
	if cfg.App.Env == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Start server with configured port
	log.Printf("Starting API server in %s mode on port %d", cfg.App.Env, cfg.App.Port)
	if err := engine.Run(":" + strconv.Itoa(cfg.App.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
