package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/api"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	database, err := db.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := api.NewRouter(database, cfg)

	// Start server
	addr := cfg.Host + ":" + cfg.Port
	log.Printf("Starting media server on %s", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
