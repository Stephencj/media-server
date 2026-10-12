package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/api"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/internal/library"
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

	// Maintenance commands run against the same database and exit, so they
	// can be used beside a running server: media-server <command>.
	if len(os.Args) > 1 {
		if err := runCommand(os.Args[1], database, cfg); err != nil {
			log.Fatalf("%s: %v", os.Args[1], err)
		}
		return
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

// runCommand runs one maintenance command. Results go to stdout as JSON; the
// log goes to stderr.
//
//	tidy-plan    what the next scan will move out of Movies (changes nothing)
//	scan         scan every source now
//	file-health  files that look like a bad rip or conversion
func runCommand(name string, database *db.DB, cfg *config.Config) error {
	out := json.NewEncoder(os.Stdout)
	out.SetIndent("", "  ")

	switch name {
	case "tidy-plan":
		steps, err := library.NewScanner(database, cfg).PlanTidy()
		if err != nil {
			return err
		}
		return out.Encode(steps)
	case "scan":
		return library.NewScanner(database, cfg).ScanAll()
	case "file-health":
		issues, err := library.LibraryFileHealth(database)
		if err != nil {
			return err
		}
		return out.Encode(issues)
	default:
		return fmt.Errorf("unknown command (want tidy-plan, scan or file-health)")
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
