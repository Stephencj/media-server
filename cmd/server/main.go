package main

import (
	"log"
	"os"
	"strings"

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

	// Seed device API keys from MEDIA_SERVER_DEVICE_KEYS env var.
	// Format: "label1=sha256hash1,label2=sha256hash2,..."
	// Idempotent: each (label, hash) inserts only if absent.
	seedDeviceKeys(database)

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

func seedDeviceKeys(database *db.DB) {
	raw := os.Getenv("MEDIA_SERVER_DEVICE_KEYS")
	if raw == "" {
		return
	}
	user, err := database.FirstUser()
	if err != nil {
		log.Printf("[device-keys] no users yet, skipping seed (will retry next start)")
		return
	}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			log.Printf("[device-keys] skipping malformed entry %q", entry)
			continue
		}
		label, hash := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if label == "" || hash == "" {
			continue
		}
		if err := database.UpsertDeviceKey(user.ID, label, hash); err != nil {
			log.Printf("[device-keys] upsert %q failed: %v", label, err)
		}
	}
}
