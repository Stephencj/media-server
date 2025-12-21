package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(1) // SQLite only supports one writer
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(time.Hour)

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS media_sources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'local',
			username TEXT,
			password TEXT,
			enabled INTEGER DEFAULT 1,
			last_scan DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS media (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			original_title TEXT,
			type TEXT NOT NULL,
			year INTEGER,
			overview TEXT,
			poster_path TEXT,
			backdrop_path TEXT,
			rating REAL,
			runtime INTEGER,
			genres TEXT,
			tmdb_id INTEGER,
			imdb_id TEXT,
			season_count INTEGER,
			episode_count INTEGER,
			source_id INTEGER,
			file_path TEXT,
			file_size INTEGER,
			duration INTEGER,
			video_codec TEXT,
			audio_codec TEXT,
			resolution TEXT,
			audio_tracks TEXT,
			subtitle_tracks TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (source_id) REFERENCES media_sources(id)
		)`,

		`CREATE TABLE IF NOT EXISTS tv_shows (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			original_title TEXT,
			year INTEGER,
			overview TEXT,
			poster_path TEXT,
			backdrop_path TEXT,
			rating REAL,
			genres TEXT,
			tmdb_id INTEGER,
			imdb_id TEXT,
			status TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS seasons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tv_show_id INTEGER NOT NULL,
			season_number INTEGER NOT NULL,
			name TEXT,
			overview TEXT,
			poster_path TEXT,
			air_date TEXT,
			episode_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (tv_show_id) REFERENCES tv_shows(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS episodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tv_show_id INTEGER NOT NULL,
			season_id INTEGER NOT NULL,
			season_number INTEGER NOT NULL,
			episode_number INTEGER NOT NULL,
			title TEXT NOT NULL,
			overview TEXT,
			still_path TEXT,
			air_date TEXT,
			runtime INTEGER,
			rating REAL,
			source_id INTEGER,
			file_path TEXT,
			file_size INTEGER,
			duration INTEGER,
			video_codec TEXT,
			audio_codec TEXT,
			resolution TEXT,
			audio_tracks TEXT,
			subtitle_tracks TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (tv_show_id) REFERENCES tv_shows(id) ON DELETE CASCADE,
			FOREIGN KEY (season_id) REFERENCES seasons(id) ON DELETE CASCADE,
			FOREIGN KEY (source_id) REFERENCES media_sources(id)
		)`,

		`CREATE TABLE IF NOT EXISTS watch_progress (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			media_id INTEGER NOT NULL,
			media_type TEXT NOT NULL,
			position INTEGER DEFAULT 0,
			duration INTEGER DEFAULT 0,
			completed INTEGER DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(user_id, media_id, media_type)
		)`,

		`CREATE TABLE IF NOT EXISTS watchlist (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			media_id INTEGER NOT NULL,
			media_type TEXT NOT NULL,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(user_id, media_id, media_type)
		)`,

		`CREATE TABLE IF NOT EXISTS playlists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS playlist_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			playlist_id INTEGER NOT NULL,
			media_id INTEGER NOT NULL,
			media_type TEXT NOT NULL,
			position INTEGER NOT NULL,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
			UNIQUE(playlist_id, media_id, media_type)
		)`,

		// Indexes for common queries
		`CREATE INDEX IF NOT EXISTS idx_media_type ON media(type)`,
		`CREATE INDEX IF NOT EXISTS idx_media_title ON media(title)`,
		`CREATE INDEX IF NOT EXISTS idx_media_source ON media(source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_episodes_show ON episodes(tv_show_id)`,
		`CREATE INDEX IF NOT EXISTS idx_episodes_season ON episodes(season_id)`,
		`CREATE INDEX IF NOT EXISTS idx_watch_progress_user ON watch_progress(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_watchlist_user ON watchlist(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_playlists_user ON playlists(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_playlist_items_playlist ON playlist_items(playlist_id)`,
	}

	for _, migration := range migrations {
		if _, err := db.conn.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
