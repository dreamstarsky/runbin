package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"runbin/internal/config"
	"runbin/internal/controller"
	"runbin/internal/repository"
	"runbin/internal/router"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadApi("config/api.yaml")

	if cfg.App.Env == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

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

	// Add CORS middleware
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	engine.Use(func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Headers", "86400")
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

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
