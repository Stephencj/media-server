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

		`CREATE TABLE IF NOT EXISTS extras (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			category TEXT NOT NULL,
			movie_id INTEGER,
			tv_show_id INTEGER,
			episode_id INTEGER,
			season_number INTEGER,
			episode_number INTEGER,
			source_id INTEGER,
			file_path TEXT UNIQUE NOT NULL,
			file_size INTEGER,
			duration INTEGER,
			video_codec TEXT,
			audio_codec TEXT,
			resolution TEXT,
			audio_tracks TEXT,
			subtitle_tracks TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (movie_id) REFERENCES media(id) ON DELETE SET NULL,
			FOREIGN KEY (tv_show_id) REFERENCES tv_shows(id) ON DELETE SET NULL,
			FOREIGN KEY (episode_id) REFERENCES episodes(id) ON DELETE SET NULL,
			FOREIGN KEY (source_id) REFERENCES media_sources(id)
		)`,

		// Customizable sections
		`CREATE TABLE IF NOT EXISTS sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			slug TEXT UNIQUE NOT NULL,
			icon TEXT,
			description TEXT,
			section_type TEXT NOT NULL DEFAULT 'standard',
			display_order INTEGER DEFAULT 0,
			is_visible BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS section_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			section_id INTEGER NOT NULL,
			field TEXT NOT NULL,
			operator TEXT NOT NULL,
			value TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (section_id) REFERENCES sections(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS media_sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			media_id INTEGER NOT NULL,
			media_type TEXT NOT NULL,
			section_id INTEGER NOT NULL,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (section_id) REFERENCES sections(id) ON DELETE CASCADE,
			UNIQUE(media_id, media_type, section_id)
		)`,

		// Channels - virtual "live TV" feature
		`CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			icon TEXT DEFAULT '📺',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS channel_sources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			source_type TEXT NOT NULL,
			source_id INTEGER,
			source_value TEXT,
			weight INTEGER DEFAULT 1,
			shuffle BOOLEAN DEFAULT 1,
			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS channel_schedule (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			media_id INTEGER NOT NULL,
			media_type TEXT NOT NULL,
			scheduled_position INTEGER NOT NULL,
			cycle_number INTEGER DEFAULT 1,
			duration INTEGER NOT NULL,
			cumulative_start INTEGER NOT NULL,
			played BOOLEAN DEFAULT 0,
			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
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
		`CREATE INDEX IF NOT EXISTS idx_extras_movie ON extras(movie_id)`,
		`CREATE INDEX IF NOT EXISTS idx_extras_tv_show ON extras(tv_show_id)`,
		`CREATE INDEX IF NOT EXISTS idx_extras_episode ON extras(episode_id)`,
		`CREATE INDEX IF NOT EXISTS idx_extras_category ON extras(category)`,
		`CREATE INDEX IF NOT EXISTS idx_sections_slug ON sections(slug)`,
		`CREATE INDEX IF NOT EXISTS idx_sections_visible ON sections(is_visible)`,
		`CREATE INDEX IF NOT EXISTS idx_sections_order ON sections(display_order)`,
		`CREATE INDEX IF NOT EXISTS idx_section_rules_section ON section_rules(section_id)`,
		`CREATE INDEX IF NOT EXISTS idx_media_sections_section ON media_sections(section_id)`,
		`CREATE INDEX IF NOT EXISTS idx_media_sections_media ON media_sections(media_id, media_type)`,
		`CREATE INDEX IF NOT EXISTS idx_channels_user ON channels(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_channel_sources_channel ON channel_sources(channel_id)`,
		`CREATE INDEX IF NOT EXISTS idx_channel_schedule_channel ON channel_schedule(channel_id, cycle_number, scheduled_position)`,

		// Insert default sections (only if sections table is empty)
		`INSERT INTO sections (name, slug, icon, section_type, display_order, is_visible)
		SELECT 'Movies', 'movies', 'film', 'smart', 1, 1
		WHERE NOT EXISTS (SELECT 1 FROM sections WHERE slug = 'movies')`,

		`INSERT INTO sections (name, slug, icon, section_type, display_order, is_visible)
		SELECT 'TV Shows', 'tv-shows', 'tv', 'smart', 2, 1
		WHERE NOT EXISTS (SELECT 1 FROM sections WHERE slug = 'tv-shows')`,

		`INSERT INTO sections (name, slug, icon, section_type, display_order, is_visible)
		SELECT 'Extras', 'extras', 'star', 'smart', 3, 1
		WHERE NOT EXISTS (SELECT 1 FROM sections WHERE slug = 'extras')`,

		// Add default rules for Movies section
		`INSERT INTO section_rules (section_id, field, operator, value)
		SELECT id, 'type', 'equals', '"movie"'
		FROM sections WHERE slug = 'movies' AND NOT EXISTS (
			SELECT 1 FROM section_rules WHERE section_id = (SELECT id FROM sections WHERE slug = 'movies')
		)`,

		// Add default rules for TV Shows section
		`INSERT INTO section_rules (section_id, field, operator, value)
		SELECT id, 'type', 'equals', '"tvshow"'
		FROM sections WHERE slug = 'tv-shows' AND NOT EXISTS (
			SELECT 1 FROM section_rules WHERE section_id = (SELECT id FROM sections WHERE slug = 'tv-shows')
		)`,

		// Add default rules for Extras section
		`INSERT INTO section_rules (section_id, field, operator, value)
		SELECT id, 'type', 'equals', '"extra"'
		FROM sections WHERE slug = 'extras' AND NOT EXISTS (
			SELECT 1 FROM section_rules WHERE section_id = (SELECT id FROM sections WHERE slug = 'extras')
		)`,
	}

	// Run migrations that might fail (e.g., column already exists)
	optionalMigrations := []string{
		// Add shuffle column to channel_sources for existing databases
		`ALTER TABLE channel_sources ADD COLUMN shuffle BOOLEAN DEFAULT 1`,
	}

	for _, migration := range optionalMigrations {
		// Ignore errors (column may already exist)
		db.conn.Exec(migration)
	}

	for _, migration := range migrations {
		if _, err := db.conn.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
