package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math/rand"
	"time"
)

var ErrNotFound = errors.New("record not found")

// ============ Generic Helper Functions ============

// Generic helper for getting a single record by ID
func getByID[T any](db *sql.DB, query string, id int64, scanner func(*sql.Row) (T, error)) (T, error) {
	row := db.QueryRow(query, id)
	return scanner(row)
}

// Generic helper for getting a single record by file path
func getByFilePath[T any](db *sql.DB, query string, path string, scanner func(*sql.Row) (T, error)) (T, error) {
	row := db.QueryRow(query, path)
	return scanner(row)
}

// Generic helper for scanning multiple rows
func scanRows[T any](rows *sql.Rows, scanner func(*sql.Rows) (T, error)) ([]T, error) {
	var results []T
	for rows.Next() {
		item, err := scanner(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

// ============ Scanner Helper Functions ============

func scanMediaRow(row *sql.Row) (Media, error) {
	var m Media
	err := row.Scan(
		&m.ID, &m.Title, &m.OriginalTitle, &m.Type, &m.Year,
		&m.Overview, &m.PosterPath, &m.BackdropPath, &m.Rating, &m.Runtime,
		&m.Genres, &m.TMDbID, &m.IMDbID, &m.SeasonCount, &m.EpisodeCount,
		&m.SourceID, &m.FilePath, &m.FileSize, &m.Duration, &m.VideoCodec,
		&m.AudioCodec, &m.Resolution, &m.AudioTracks, &m.SubtitleTracks,
		&m.CreatedAt, &m.UpdatedAt,
	)
	return m, err
}

func scanEpisodeRow(row *sql.Row) (Episode, error) {
	var e Episode
	err := row.Scan(
		&e.ID, &e.TVShowID, &e.SeasonID, &e.SeasonNumber,
		&e.EpisodeNumber, &e.Title, &e.Overview, &e.StillPath,
		&e.AirDate, &e.Runtime, &e.Rating, &e.SourceID, &e.FilePath,
		&e.FileSize, &e.Duration, &e.VideoCodec, &e.AudioCodec,
		&e.Resolution, &e.AudioTracks, &e.SubtitleTracks,
		&e.CreatedAt, &e.UpdatedAt,
	)
	return e, err
}

func scanExtraRow(row *sql.Row) (Extra, error) {
	var ex Extra
	err := row.Scan(
		&ex.ID, &ex.Title, &ex.Category, &ex.MovieID, &ex.TVShowID, &ex.EpisodeID,
		&ex.SeasonNumber, &ex.EpisodeNumber, &ex.SourceID, &ex.FilePath, &ex.FileSize,
		&ex.Duration, &ex.VideoCodec, &ex.AudioCodec, &ex.Resolution,
		&ex.AudioTracks, &ex.SubtitleTracks, &ex.CreatedAt, &ex.UpdatedAt,
	)
	return ex, err
}

// User Repository Methods

// CreateUser creates a new user
func (db *DB) CreateUser(username, email, passwordHash string) (*User, error) {
	result, err := db.conn.Exec(
		`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
		username, email, passwordHash,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetUserByID(id)
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := db.conn.QueryRow(
		`SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return user, err
}

// GetUserByUsername retrieves a user by username
func (db *DB) GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := db.conn.QueryRow(
		`SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return user, err
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	err := db.conn.QueryRow(
		`SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = ?`,
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return user, err
}

// Media Source Repository Methods

// CreateMediaSource creates a new media source
func (db *DB) CreateMediaSource(source *MediaSource) (*MediaSource, error) {
	result, err := db.conn.Exec(
		`INSERT INTO media_sources (name, path, type, username, password, enabled) VALUES (?, ?, ?, ?, ?, ?)`,
		source.Name, source.Path, source.Type, source.Username, source.Password, source.Enabled,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetMediaSourceByID(id)
}

// GetMediaSourceByID retrieves a media source by ID
func (db *DB) GetMediaSourceByID(id int64) (*MediaSource, error) {
	source := &MediaSource{}
	var lastScan sql.NullTime
	err := db.conn.QueryRow(
		`SELECT id, name, path, type, username, password, enabled, last_scan, created_at, updated_at
		 FROM media_sources WHERE id = ?`,
		id,
	).Scan(&source.ID, &source.Name, &source.Path, &source.Type, &source.Username,
		&source.Password, &source.Enabled, &lastScan, &source.CreatedAt, &source.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if lastScan.Valid {
		source.LastScan = lastScan.Time
	}
	return source, err
}

// GetAllMediaSources retrieves all media sources
func (db *DB) GetAllMediaSources() ([]*MediaSource, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, path, type, username, password, enabled, last_scan, created_at, updated_at
		 FROM media_sources ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*MediaSource
	for rows.Next() {
		source := &MediaSource{}
		var lastScan sql.NullTime
		if err := rows.Scan(&source.ID, &source.Name, &source.Path, &source.Type,
			&source.Username, &source.Password, &source.Enabled, &lastScan,
			&source.CreatedAt, &source.UpdatedAt); err != nil {
			return nil, err
		}
		if lastScan.Valid {
			source.LastScan = lastScan.Time
		}
		sources = append(sources, source)
	}
	return sources, nil
}

// DeleteMediaSource deletes a media source
func (db *DB) DeleteMediaSource(id int64) error {
	result, err := db.conn.Exec(`DELETE FROM media_sources WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateMediaSourceLastScan updates the last scan time
func (db *DB) UpdateMediaSourceLastScan(id int64) error {
	_, err := db.conn.Exec(
		`UPDATE media_sources SET last_scan = ?, updated_at = ? WHERE id = ?`,
		time.Now(), time.Now(), id,
	)
	return err
}

// Media Repository Methods

// CreateMedia creates a new media item
func (db *DB) CreateMedia(media *Media) (*Media, error) {
	result, err := db.conn.Exec(
		`INSERT INTO media (title, original_title, type, year, overview, poster_path, backdrop_path,
			rating, runtime, genres, tmdb_id, imdb_id, season_count, episode_count, source_id,
			file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		media.Title, media.OriginalTitle, media.Type, media.Year, media.Overview,
		media.PosterPath, media.BackdropPath, media.Rating, media.Runtime, media.Genres,
		media.TMDbID, media.IMDbID, media.SeasonCount, media.EpisodeCount, media.SourceID,
		media.FilePath, media.FileSize, media.Duration, media.VideoCodec, media.AudioCodec,
		media.Resolution, media.AudioTracks, media.SubtitleTracks,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetMediaByID(id)
}

// GetMediaByID retrieves media by ID
func (db *DB) GetMediaByID(id int64) (*Media, error) {
	query := `SELECT id, title, original_title, type, year, overview, poster_path, backdrop_path,
		rating, runtime, genres, tmdb_id, imdb_id, season_count, episode_count, source_id,
		file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks,
		subtitle_tracks, created_at, updated_at
	 FROM media WHERE id = ?`
	media, err := getByID(db.conn, query, id, scanMediaRow)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &media, nil
}

// GetMediaByType retrieves all media of a specific type
func (db *DB) GetMediaByType(mediaType MediaType, limit, offset int) ([]*Media, error) {
	rows, err := db.conn.Query(
		`SELECT id, title, original_title, type, year, overview, poster_path, backdrop_path,
			rating, runtime, genres, tmdb_id, imdb_id, season_count, episode_count, source_id,
			file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks,
			subtitle_tracks, created_at, updated_at
		 FROM media WHERE type = ? ORDER BY title LIMIT ? OFFSET ?`,
		mediaType, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMediaRows(rows)
}

// GetRecentMedia retrieves recently added media
func (db *DB) GetRecentMedia(limit int) ([]*Media, error) {
	rows, err := db.conn.Query(
		`SELECT id, title, original_title, type, year, overview, poster_path, backdrop_path,
			rating, runtime, genres, tmdb_id, imdb_id, season_count, episode_count, source_id,
			file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks,
			subtitle_tracks, created_at, updated_at
		 FROM media ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMediaRows(rows)
}

// GetMediaByFilePath checks if media with given file path exists
func (db *DB) GetMediaByFilePath(filePath string) (*Media, error) {
	query := `SELECT id, title, original_title, type, year, overview, poster_path, backdrop_path,
		rating, runtime, genres, tmdb_id, imdb_id, season_count, episode_count, source_id,
		file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks,
		subtitle_tracks, created_at, updated_at
	 FROM media WHERE file_path = ?`
	media, err := getByFilePath(db.conn, query, filePath, scanMediaRow)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func scanMediaRows(rows *sql.Rows) ([]*Media, error) {
	items := make([]*Media, 0) // Initialize as empty slice, not nil (ensures JSON [] not null)
	for rows.Next() {
		media := &Media{}
		if err := rows.Scan(&media.ID, &media.Title, &media.OriginalTitle, &media.Type,
			&media.Year, &media.Overview, &media.PosterPath, &media.BackdropPath, &media.Rating,
			&media.Runtime, &media.Genres, &media.TMDbID, &media.IMDbID, &media.SeasonCount,
			&media.EpisodeCount, &media.SourceID, &media.FilePath, &media.FileSize, &media.Duration,
			&media.VideoCodec, &media.AudioCodec, &media.Resolution, &media.AudioTracks,
			&media.SubtitleTracks, &media.CreatedAt, &media.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, media)
	}
	return items, nil
}

// SearchMedia searches for media by title with fuzzy matching
func (db *DB) SearchMedia(query string, mediaType MediaType, limit int) ([]*Media, error) {
	rows, err := db.conn.Query(
		`SELECT id, title, original_title, type, year, overview, poster_path, backdrop_path,
			rating, runtime, genres, tmdb_id, imdb_id, season_count, episode_count, source_id,
			file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks,
			subtitle_tracks, created_at, updated_at
		 FROM media WHERE type = ? AND (title LIKE ? OR original_title LIKE ?)
		 ORDER BY title LIMIT ?`,
		mediaType, "%"+query+"%", "%"+query+"%", limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMediaRows(rows)
}

// Watch Progress Repository Methods

// UpsertWatchProgress creates or updates watch progress
func (db *DB) UpsertWatchProgress(userID, mediaID int64, mediaType MediaType, position, duration int, completed bool) error {
	_, err := db.conn.Exec(
		`INSERT INTO watch_progress (user_id, media_id, media_type, position, duration, completed, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, media_id, media_type) DO UPDATE SET
		 position = excluded.position, duration = excluded.duration,
		 completed = excluded.completed, updated_at = excluded.updated_at`,
		userID, mediaID, mediaType, position, duration, completed, time.Now(),
	)
	return err
}

// GetWatchProgress retrieves watch progress for a user and media
func (db *DB) GetWatchProgress(userID, mediaID int64, mediaType MediaType) (*WatchProgress, error) {
	progress := &WatchProgress{}
	err := db.conn.QueryRow(
		`SELECT id, user_id, media_id, media_type, position, duration, completed, updated_at
		 FROM watch_progress WHERE user_id = ? AND media_id = ? AND media_type = ?`,
		userID, mediaID, mediaType,
	).Scan(&progress.ID, &progress.UserID, &progress.MediaID, &progress.MediaType,
		&progress.Position, &progress.Duration, &progress.Completed, &progress.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return progress, err
}

// GetContinueWatching retrieves in-progress media for a user
func (db *DB) GetContinueWatching(userID int64, limit int) ([]*WatchProgress, error) {
	rows, err := db.conn.Query(
		`SELECT id, user_id, media_id, media_type, position, duration, completed, updated_at
		 FROM watch_progress
		 WHERE user_id = ? AND completed = 0 AND position > 0
		 ORDER BY updated_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*WatchProgress
	for rows.Next() {
		p := &WatchProgress{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.MediaID, &p.MediaType,
			&p.Position, &p.Duration, &p.Completed, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, nil
}

// Watchlist Repository Methods

// AddToWatchlist adds a media item to user's watchlist
func (db *DB) AddToWatchlist(userID, mediaID int64, mediaType MediaType) error {
	_, err := db.conn.Exec(
		`INSERT OR IGNORE INTO watchlist (user_id, media_id, media_type, added_at)
		 VALUES (?, ?, ?, ?)`,
		userID, mediaID, mediaType, time.Now(),
	)
	return err
}

// RemoveFromWatchlist removes a media item from user's watchlist
func (db *DB) RemoveFromWatchlist(userID, mediaID int64, mediaType MediaType) error {
	_, err := db.conn.Exec(
		`DELETE FROM watchlist WHERE user_id = ? AND media_id = ? AND media_type = ?`,
		userID, mediaID, mediaType,
	)
	return err
}

// IsInWatchlist checks if a media item is in user's watchlist
func (db *DB) IsInWatchlist(userID, mediaID int64, mediaType MediaType) (bool, error) {
	var count int
	err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM watchlist WHERE user_id = ? AND media_id = ? AND media_type = ?`,
		userID, mediaID, mediaType,
	).Scan(&count)
	return count > 0, err
}

// GetWatchlist retrieves user's watchlist with media details
func (db *DB) GetWatchlist(userID int64, limit int) ([]*Media, error) {
	rows, err := db.conn.Query(
		`SELECT m.id, m.title, m.original_title, m.type, m.year, m.overview, m.poster_path, m.backdrop_path,
			m.rating, m.runtime, m.genres, m.tmdb_id, m.imdb_id, m.season_count, m.episode_count, m.source_id,
			m.file_path, m.file_size, m.duration, m.video_codec, m.audio_codec, m.resolution, m.audio_tracks,
			m.subtitle_tracks, m.created_at, m.updated_at
		 FROM watchlist w
		 JOIN media m ON w.media_id = m.id
		 WHERE w.user_id = ?
		 ORDER BY w.added_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMediaRows(rows)
}

// UpdateMedia updates an existing media item
func (db *DB) UpdateMedia(media *Media) error {
	_, err := db.conn.Exec(
		`UPDATE media SET
			title = ?, original_title = ?, overview = ?, poster_path = ?, backdrop_path = ?,
			rating = ?, runtime = ?, genres = ?, tmdb_id = ?, imdb_id = ?,
			season_count = ?, episode_count = ?, year = ?, updated_at = ?
		 WHERE id = ?`,
		media.Title, media.OriginalTitle, media.Overview, media.PosterPath, media.BackdropPath,
		media.Rating, media.Runtime, media.Genres, media.TMDbID, media.IMDbID,
		media.SeasonCount, media.EpisodeCount, media.Year, time.Now(), media.ID,
	)
	return err
}

// MarkAsWatched marks a media item as completed (100% watched)
func (db *DB) MarkAsWatched(userID, mediaID int64, mediaType MediaType) error {
	// Get media duration if available
	var duration int
	db.conn.QueryRow(`SELECT COALESCE(duration, 0) FROM media WHERE id = ?`, mediaID).Scan(&duration)

	_, err := db.conn.Exec(
		`INSERT INTO watch_progress (user_id, media_id, media_type, position, duration, completed, updated_at)
		 VALUES (?, ?, ?, ?, ?, 1, ?)
		 ON CONFLICT(user_id, media_id, media_type) DO UPDATE SET
		 completed = 1, updated_at = excluded.updated_at`,
		userID, mediaID, mediaType, duration, duration, time.Now(),
	)
	return err
}

// Playlist Repository Methods

// CreatePlaylist creates a new playlist
func (db *DB) CreatePlaylist(userID int64, name, description string) (*Playlist, error) {
	result, err := db.conn.Exec(
		`INSERT INTO playlists (user_id, name, description) VALUES (?, ?, ?)`,
		userID, name, description,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetPlaylistByID(id)
}

// GetPlaylistByID retrieves a playlist by ID with item count
func (db *DB) GetPlaylistByID(id int64) (*Playlist, error) {
	playlist := &Playlist{}
	err := db.conn.QueryRow(
		`SELECT p.id, p.user_id, p.name, p.description, p.is_public, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM playlist_items WHERE playlist_id = p.id) as item_count
		 FROM playlists p WHERE p.id = ?`,
		id,
	).Scan(&playlist.ID, &playlist.UserID, &playlist.Name, &playlist.Description,
		&playlist.IsPublic, &playlist.CreatedAt, &playlist.UpdatedAt, &playlist.ItemCount)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return playlist, err
}

// GetUserPlaylists retrieves all playlists for a user (including public playlists)
func (db *DB) GetUserPlaylists(userID int64) ([]*Playlist, error) {
	rows, err := db.conn.Query(
		`SELECT p.id, p.user_id, p.name, p.description, p.is_public, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM playlist_items WHERE playlist_id = p.id) as item_count
		 FROM playlists p
		 WHERE p.user_id = ? OR p.is_public = 1
		 ORDER BY p.user_id = ? DESC, p.updated_at DESC`,
		userID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	playlists := make([]*Playlist, 0)
	for rows.Next() {
		p := &Playlist{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description,
			&p.IsPublic, &p.CreatedAt, &p.UpdatedAt, &p.ItemCount); err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}
	return playlists, nil
}

// UpdatePlaylist updates a playlist's name and description
func (db *DB) UpdatePlaylist(id int64, name, description string) error {
	result, err := db.conn.Exec(
		`UPDATE playlists SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		name, description, time.Now(), id,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeletePlaylist deletes a playlist and all its items
func (db *DB) DeletePlaylist(id int64) error {
	result, err := db.conn.Exec(`DELETE FROM playlists WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// AddToPlaylist adds a media item to a playlist
func (db *DB) AddToPlaylist(playlistID, mediaID int64, mediaType MediaType) error {
	// Get the next position
	var maxPos int
	db.conn.QueryRow(
		`SELECT COALESCE(MAX(position), 0) FROM playlist_items WHERE playlist_id = ?`,
		playlistID,
	).Scan(&maxPos)

	_, err := db.conn.Exec(
		`INSERT INTO playlist_items (playlist_id, media_id, media_type, position)
		 VALUES (?, ?, ?, ?)`,
		playlistID, mediaID, mediaType, maxPos+1,
	)
	if err != nil {
		return err
	}

	// Update playlist's updated_at timestamp
	db.conn.Exec(`UPDATE playlists SET updated_at = ? WHERE id = ?`, time.Now(), playlistID)
	return nil
}

// RemoveFromPlaylist removes a media item from a playlist
func (db *DB) RemoveFromPlaylist(playlistID, mediaID int64, mediaType MediaType) error {
	result, err := db.conn.Exec(
		`DELETE FROM playlist_items WHERE playlist_id = ? AND media_id = ? AND media_type = ?`,
		playlistID, mediaID, mediaType,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	// Reorder remaining items
	db.conn.Exec(
		`UPDATE playlist_items SET position = (
			SELECT COUNT(*) FROM playlist_items pi2
			WHERE pi2.playlist_id = playlist_items.playlist_id
			AND pi2.id <= playlist_items.id
		) WHERE playlist_id = ?`,
		playlistID,
	)

	// Update playlist's updated_at timestamp
	db.conn.Exec(`UPDATE playlists SET updated_at = ? WHERE id = ?`, time.Now(), playlistID)
	return nil
}

// GetPlaylistItems retrieves all items in a playlist with media details
func (db *DB) GetPlaylistItems(playlistID int64) ([]*PlaylistItemWithMedia, error) {
	// Use UNION to get items from both media table (movies) and episodes table
	rows, err := db.conn.Query(
		`SELECT pi.id, pi.playlist_id, pi.media_id, pi.media_type, pi.position, pi.added_at,
			m.title, m.year, m.poster_path, m.duration, m.overview, m.rating, m.resolution
		 FROM playlist_items pi
		 JOIN media m ON pi.media_id = m.id
		 WHERE pi.playlist_id = ? AND pi.media_type = 'movie'

		 UNION ALL

		 SELECT pi.id, pi.playlist_id, pi.media_id, pi.media_type, pi.position, pi.added_at,
			e.title, 0 as year, e.still_path as poster_path, e.duration, e.overview, e.rating, e.resolution
		 FROM playlist_items pi
		 JOIN episodes e ON pi.media_id = e.id
		 WHERE pi.playlist_id = ? AND pi.media_type = 'episode'

		 ORDER BY position ASC`,
		playlistID, playlistID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*PlaylistItemWithMedia, 0)
	for rows.Next() {
		item := &PlaylistItemWithMedia{}
		if err := rows.Scan(&item.ID, &item.PlaylistID, &item.MediaID, &item.MediaType,
			&item.Position, &item.AddedAt, &item.Title, &item.Year, &item.PosterPath,
			&item.Duration, &item.Overview, &item.Rating, &item.Resolution); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// ReorderPlaylistItems reorders items in a playlist based on the provided order
func (db *DB) ReorderPlaylistItems(playlistID int64, itemIDs []int64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, itemID := range itemIDs {
		_, err := tx.Exec(
			`UPDATE playlist_items SET position = ? WHERE id = ? AND playlist_id = ?`,
			i+1, itemID, playlistID,
		)
		if err != nil {
			return err
		}
	}

	// Update playlist's updated_at timestamp
	tx.Exec(`UPDATE playlists SET updated_at = ? WHERE id = ?`, time.Now(), playlistID)

	return tx.Commit()
}

// ============ TV Show Repository Methods ============

// CreateTVShow creates a new TV show
func (db *DB) CreateTVShow(show *TVShow) (*TVShow, error) {
	result, err := db.conn.Exec(
		`INSERT INTO tv_shows (title, original_title, year, overview, poster_path, backdrop_path,
			rating, genres, tmdb_id, imdb_id, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		show.Title, show.OriginalTitle, show.Year, show.Overview, show.PosterPath,
		show.BackdropPath, show.Rating, show.Genres, show.TMDbID, show.IMDbID, show.Status,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetTVShowByID(id)
}

// GetTVShowByID retrieves a TV show by ID with aggregated metadata
func (db *DB) GetTVShowByID(id int64) (*TVShow, error) {
	query := `
		SELECT
			s.id, s.title, COALESCE(s.original_title, ''), s.year, COALESCE(s.overview, ''),
			COALESCE(s.poster_path, ''), COALESCE(s.backdrop_path, ''), s.rating, COALESCE(s.genres, ''),
			s.tmdb_id, COALESCE(s.imdb_id, ''), COALESCE(s.status, ''), s.created_at, s.updated_at,
			COUNT(DISTINCT se.id) as season_count,
			COUNT(DISTINCT e.id) as episode_count,
			(SELECT resolution FROM episodes WHERE tv_show_id = s.id
			 GROUP BY resolution ORDER BY COUNT(*) DESC LIMIT 1) as common_resolution,
			(SELECT video_codec FROM episodes WHERE tv_show_id = s.id
			 GROUP BY video_codec ORDER BY COUNT(*) DESC LIMIT 1) as common_video_codec,
			(SELECT audio_codec FROM episodes WHERE tv_show_id = s.id
			 GROUP BY audio_codec ORDER BY COUNT(*) DESC LIMIT 1) as common_audio_codec,
			CAST(COALESCE(SUM(e.duration), 0) AS INTEGER) as total_duration,
			CAST(COALESCE(AVG(e.duration), 0) AS INTEGER) as avg_episode_length,
			(SELECT resolution FROM episodes WHERE tv_show_id = s.id
			 ORDER BY CAST(SUBSTR(resolution, 1, INSTR(resolution, 'x')-1) AS INTEGER) DESC
			 LIMIT 1) as max_resolution
		FROM tv_shows s
		LEFT JOIN seasons se ON se.tv_show_id = s.id
		LEFT JOIN episodes e ON e.tv_show_id = s.id
		WHERE s.id = ?
		GROUP BY s.id
	`

	show := &TVShow{}
	var commonResolution, commonVideoCodec, commonAudioCodec, maxResolution sql.NullString
	err := db.conn.QueryRow(query, id).Scan(
		&show.ID, &show.Title, &show.OriginalTitle, &show.Year, &show.Overview,
		&show.PosterPath, &show.BackdropPath, &show.Rating, &show.Genres,
		&show.TMDbID, &show.IMDbID, &show.Status, &show.CreatedAt, &show.UpdatedAt,
		&show.SeasonCount, &show.EpisodeCount,
		&commonResolution, &commonVideoCodec, &commonAudioCodec,
		&show.TotalDuration, &show.AvgEpisodeLength, &maxResolution,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if commonResolution.Valid {
		show.CommonResolution = commonResolution.String
	}
	if commonVideoCodec.Valid {
		show.CommonVideoCodec = commonVideoCodec.String
	}
	if commonAudioCodec.Valid {
		show.CommonAudioCodec = commonAudioCodec.String
	}
	if maxResolution.Valid {
		show.MaxResolution = maxResolution.String
	}

	return show, nil
}

// GetTVShowByTMDBID retrieves a TV show by TMDB ID
func (db *DB) GetTVShowByTMDBID(tmdbID int) (*TVShow, error) {
	show := &TVShow{}
	err := db.conn.QueryRow(
		`SELECT id, title, original_title, year, overview, poster_path, backdrop_path,
			rating, genres, tmdb_id, imdb_id, status, created_at, updated_at
		 FROM tv_shows WHERE tmdb_id = ?`,
		tmdbID,
	).Scan(&show.ID, &show.Title, &show.OriginalTitle, &show.Year, &show.Overview,
		&show.PosterPath, &show.BackdropPath, &show.Rating, &show.Genres, &show.TMDbID,
		&show.IMDbID, &show.Status, &show.CreatedAt, &show.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return show, err
}

// GetTVShowByTitle retrieves a TV show by title
func (db *DB) GetTVShowByTitle(title string) (*TVShow, error) {
	show := &TVShow{}
	err := db.conn.QueryRow(
		`SELECT id, title, original_title, year, overview, poster_path, backdrop_path,
			rating, genres, tmdb_id, imdb_id, status, created_at, updated_at
		 FROM tv_shows WHERE title = ? COLLATE NOCASE`,
		title,
	).Scan(&show.ID, &show.Title, &show.OriginalTitle, &show.Year, &show.Overview,
		&show.PosterPath, &show.BackdropPath, &show.Rating, &show.Genres, &show.TMDbID,
		&show.IMDbID, &show.Status, &show.CreatedAt, &show.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return show, err
}

// GetAllTVShows retrieves all TV shows with season/episode counts and aggregated metadata
func (db *DB) GetAllTVShows(limit, offset int) ([]*TVShow, int, error) {
	// Get total count
	var total int
	db.conn.QueryRow(`SELECT COUNT(*) FROM tv_shows`).Scan(&total)

	query := `
		SELECT
			s.id, s.title, COALESCE(s.original_title, ''), s.year, COALESCE(s.overview, ''),
			COALESCE(s.poster_path, ''), COALESCE(s.backdrop_path, ''), s.rating, COALESCE(s.genres, ''),
			s.tmdb_id, COALESCE(s.imdb_id, ''), COALESCE(s.status, ''), s.created_at, s.updated_at,
			COUNT(DISTINCT se.id) as season_count,
			COUNT(DISTINCT e.id) as episode_count,
			(SELECT resolution FROM episodes WHERE tv_show_id = s.id
			 GROUP BY resolution ORDER BY COUNT(*) DESC LIMIT 1) as common_resolution,
			(SELECT video_codec FROM episodes WHERE tv_show_id = s.id
			 GROUP BY video_codec ORDER BY COUNT(*) DESC LIMIT 1) as common_video_codec,
			(SELECT audio_codec FROM episodes WHERE tv_show_id = s.id
			 GROUP BY audio_codec ORDER BY COUNT(*) DESC LIMIT 1) as common_audio_codec,
			CAST(COALESCE(SUM(e.duration), 0) AS INTEGER) as total_duration,
			CAST(COALESCE(AVG(e.duration), 0) AS INTEGER) as avg_episode_length,
			(SELECT resolution FROM episodes WHERE tv_show_id = s.id
			 ORDER BY CAST(SUBSTR(resolution, 1, INSTR(resolution, 'x')-1) AS INTEGER) DESC
			 LIMIT 1) as max_resolution
		FROM tv_shows s
		LEFT JOIN seasons se ON se.tv_show_id = s.id
		LEFT JOIN episodes e ON e.tv_show_id = s.id
		GROUP BY s.id
		ORDER BY s.title
		LIMIT ? OFFSET ?
	`

	rows, err := db.conn.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	shows := make([]*TVShow, 0)
	for rows.Next() {
		show := &TVShow{}
		var commonResolution, commonVideoCodec, commonAudioCodec, maxResolution sql.NullString
		if err := rows.Scan(
			&show.ID, &show.Title, &show.OriginalTitle, &show.Year, &show.Overview,
			&show.PosterPath, &show.BackdropPath, &show.Rating, &show.Genres,
			&show.TMDbID, &show.IMDbID, &show.Status, &show.CreatedAt, &show.UpdatedAt,
			&show.SeasonCount, &show.EpisodeCount,
			&commonResolution, &commonVideoCodec, &commonAudioCodec,
			&show.TotalDuration, &show.AvgEpisodeLength, &maxResolution,
		); err != nil {
			return nil, 0, err
		}

		// Handle nullable fields
		if commonResolution.Valid {
			show.CommonResolution = commonResolution.String
		}
		if commonVideoCodec.Valid {
			show.CommonVideoCodec = commonVideoCodec.String
		}
		if commonAudioCodec.Valid {
			show.CommonAudioCodec = commonAudioCodec.String
		}
		if maxResolution.Valid {
			show.MaxResolution = maxResolution.String
		}

		shows = append(shows, show)
	}
	return shows, total, nil
}

// UpdateTVShow updates a TV show
func (db *DB) UpdateTVShow(show *TVShow) error {
	_, err := db.conn.Exec(
		`UPDATE tv_shows SET title = ?, original_title = ?, year = ?, overview = ?,
			poster_path = ?, backdrop_path = ?, rating = ?, genres = ?, tmdb_id = ?,
			imdb_id = ?, status = ?, updated_at = ?
		 WHERE id = ?`,
		show.Title, show.OriginalTitle, show.Year, show.Overview, show.PosterPath,
		show.BackdropPath, show.Rating, show.Genres, show.TMDbID, show.IMDbID,
		show.Status, time.Now(), show.ID,
	)
	return err
}

// SearchTVShows searches for TV shows by title with fuzzy matching
func (db *DB) SearchTVShows(query string, limit int) ([]*TVShow, error) {
	rows, err := db.conn.Query(
		`SELECT t.id, t.title, t.original_title, t.year, t.overview, t.poster_path, t.backdrop_path,
			t.rating, t.genres, t.tmdb_id, t.imdb_id, t.status, t.created_at, t.updated_at,
			(SELECT COUNT(DISTINCT season_number) FROM episodes WHERE tv_show_id = t.id) as season_count,
			(SELECT COUNT(*) FROM episodes WHERE tv_show_id = t.id) as episode_count
		 FROM tv_shows t WHERE t.title LIKE ? OR t.original_title LIKE ?
		 ORDER BY t.title LIMIT ?`,
		"%"+query+"%", "%"+query+"%", limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shows := make([]*TVShow, 0)
	for rows.Next() {
		show := &TVShow{}
		if err := rows.Scan(&show.ID, &show.Title, &show.OriginalTitle, &show.Year, &show.Overview,
			&show.PosterPath, &show.BackdropPath, &show.Rating, &show.Genres, &show.TMDbID,
			&show.IMDbID, &show.Status, &show.CreatedAt, &show.UpdatedAt,
			&show.SeasonCount, &show.EpisodeCount); err != nil {
			return nil, err
		}
		shows = append(shows, show)
	}
	return shows, nil
}

// SearchTVShowsFuzzy searches for TV shows with bidirectional fuzzy matching
// This allows "Psych Commentary" to match "Psych" by checking if query CONTAINS show title
func (db *DB) SearchTVShowsFuzzy(query string, limit int) ([]*TVShow, error) {
	// First try standard search (show title contains query)
	shows, err := db.SearchTVShows(query, limit)
	if err == nil && len(shows) > 0 {
		return shows, nil
	}

	// If no results, try reverse match: find shows where query CONTAINS the show title
	// This handles "Psych Commentary" matching "Psych"
	rows, err := db.conn.Query(
		`SELECT t.id, t.title, t.original_title, t.year, t.overview, t.poster_path, t.backdrop_path,
			t.rating, t.genres, t.tmdb_id, t.imdb_id, t.status, t.created_at, t.updated_at,
			(SELECT COUNT(DISTINCT season_number) FROM episodes WHERE tv_show_id = t.id) as season_count,
			(SELECT COUNT(*) FROM episodes WHERE tv_show_id = t.id) as episode_count
		 FROM tv_shows t
		 WHERE ? LIKE '%' || t.title || '%' COLLATE NOCASE
		    OR ? LIKE '%' || t.original_title || '%' COLLATE NOCASE
		 ORDER BY LENGTH(t.title) DESC
		 LIMIT ?`,
		query, query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shows = make([]*TVShow, 0)
	for rows.Next() {
		show := &TVShow{}
		if err := rows.Scan(&show.ID, &show.Title, &show.OriginalTitle, &show.Year, &show.Overview,
			&show.PosterPath, &show.BackdropPath, &show.Rating, &show.Genres, &show.TMDbID,
			&show.IMDbID, &show.Status, &show.CreatedAt, &show.UpdatedAt,
			&show.SeasonCount, &show.EpisodeCount); err != nil {
			return nil, err
		}
		shows = append(shows, show)
	}
	return shows, nil
}

// SearchMediaFuzzy searches for media with bidirectional fuzzy matching
func (db *DB) SearchMediaFuzzy(query string, mediaType MediaType, limit int) ([]*Media, error) {
	// First try standard search
	media, err := db.SearchMedia(query, mediaType, limit)
	if err == nil && len(media) > 0 {
		return media, nil
	}

	// Try reverse match: query CONTAINS title
	rows, err := db.conn.Query(
		`SELECT id, title, original_title, type, year, overview, poster_path, backdrop_path,
			rating, runtime, genres, tmdb_id, imdb_id, season_count, episode_count, source_id,
			file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks,
			subtitle_tracks, created_at, updated_at
		 FROM media
		 WHERE type = ? AND (? LIKE '%' || title || '%' COLLATE NOCASE
		    OR ? LIKE '%' || original_title || '%' COLLATE NOCASE)
		 ORDER BY LENGTH(title) DESC LIMIT ?`,
		mediaType, query, query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMediaRows(rows)
}

// ============ Season Repository Methods ============

// CreateSeason creates a new season
func (db *DB) CreateSeason(season *Season) (*Season, error) {
	result, err := db.conn.Exec(
		`INSERT INTO seasons (tv_show_id, season_number, name, overview, poster_path, air_date, episode_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		season.TVShowID, season.SeasonNumber, season.Name, season.Overview,
		season.PosterPath, season.AirDate, season.EpisodeCount,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetSeasonByID(id)
}

// GetSeasonByID retrieves a season by ID
func (db *DB) GetSeasonByID(id int64) (*Season, error) {
	season := &Season{}
	err := db.conn.QueryRow(
		`SELECT id, tv_show_id, season_number, name, overview, poster_path, air_date, episode_count, created_at
		 FROM seasons WHERE id = ?`,
		id,
	).Scan(&season.ID, &season.TVShowID, &season.SeasonNumber, &season.Name, &season.Overview,
		&season.PosterPath, &season.AirDate, &season.EpisodeCount, &season.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return season, err
}

// GetSeasonByNumber retrieves a season by show ID and season number
func (db *DB) GetSeasonByNumber(showID int64, seasonNum int) (*Season, error) {
	season := &Season{}
	err := db.conn.QueryRow(
		`SELECT id, tv_show_id, season_number, name, overview, poster_path, air_date, episode_count, created_at
		 FROM seasons WHERE tv_show_id = ? AND season_number = ?`,
		showID, seasonNum,
	).Scan(&season.ID, &season.TVShowID, &season.SeasonNumber, &season.Name, &season.Overview,
		&season.PosterPath, &season.AirDate, &season.EpisodeCount, &season.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return season, err
}

// GetSeasonsByShowID retrieves all seasons for a TV show
func (db *DB) GetSeasonsByShowID(showID int64) ([]*Season, error) {
	rows, err := db.conn.Query(
		`SELECT s.id, s.tv_show_id, s.season_number, s.name, s.overview, s.poster_path, s.air_date,
			s.episode_count, s.created_at,
			(SELECT COUNT(*) FROM episodes WHERE season_id = s.id) as actual_episode_count
		 FROM seasons s WHERE s.tv_show_id = ? ORDER BY s.season_number`,
		showID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seasons := make([]*Season, 0)
	for rows.Next() {
		season := &Season{}
		var actualCount int
		if err := rows.Scan(&season.ID, &season.TVShowID, &season.SeasonNumber, &season.Name,
			&season.Overview, &season.PosterPath, &season.AirDate, &season.EpisodeCount,
			&season.CreatedAt, &actualCount); err != nil {
			return nil, err
		}
		// Use actual episode count from database if we have episodes
		if actualCount > 0 {
			season.EpisodeCount = actualCount
		}
		seasons = append(seasons, season)
	}
	return seasons, nil
}

// ============ Episode Repository Methods ============

// CreateEpisode creates a new episode
func (db *DB) CreateEpisode(episode *Episode) (*Episode, error) {
	result, err := db.conn.Exec(
		`INSERT INTO episodes (tv_show_id, season_id, season_number, episode_number, title, overview,
			still_path, air_date, runtime, rating, source_id, file_path, file_size, duration,
			video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		episode.TVShowID, episode.SeasonID, episode.SeasonNumber, episode.EpisodeNumber,
		episode.Title, episode.Overview, episode.StillPath, episode.AirDate, episode.Runtime,
		episode.Rating, episode.SourceID, episode.FilePath, episode.FileSize, episode.Duration,
		episode.VideoCodec, episode.AudioCodec, episode.Resolution, episode.AudioTracks,
		episode.SubtitleTracks,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetEpisodeByID(id)
}

// GetEpisodeByID retrieves an episode by ID
func (db *DB) GetEpisodeByID(id int64) (*Episode, error) {
	query := `SELECT id, tv_show_id, season_id, season_number, episode_number, title, overview,
		still_path, air_date, runtime, rating, source_id, file_path, file_size, duration,
		video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks, created_at, updated_at
	 FROM episodes WHERE id = ?`
	episode, err := getByID(db.conn, query, id, scanEpisodeRow)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &episode, nil
}

// GetEpisodeByFilePath retrieves an episode by file path
func (db *DB) GetEpisodeByFilePath(filePath string) (*Episode, error) {
	query := `SELECT id, tv_show_id, season_id, season_number, episode_number, title, overview,
		still_path, air_date, runtime, rating, source_id, file_path, file_size, duration,
		video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks, created_at, updated_at
	 FROM episodes WHERE file_path = ?`
	episode, err := getByFilePath(db.conn, query, filePath, scanEpisodeRow)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &episode, nil
}

// GetEpisodeByNumber retrieves an episode by show ID, season, and episode number
func (db *DB) GetEpisodeByNumber(showID int64, seasonNum, episodeNum int) (*Episode, error) {
	episode := &Episode{}
	err := db.conn.QueryRow(
		`SELECT id, tv_show_id, season_id, season_number, episode_number, title, overview,
			still_path, air_date, runtime, rating, source_id, file_path, file_size, duration,
			video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM episodes WHERE tv_show_id = ? AND season_number = ? AND episode_number = ?`,
		showID, seasonNum, episodeNum,
	).Scan(&episode.ID, &episode.TVShowID, &episode.SeasonID, &episode.SeasonNumber,
		&episode.EpisodeNumber, &episode.Title, &episode.Overview, &episode.StillPath,
		&episode.AirDate, &episode.Runtime, &episode.Rating, &episode.SourceID, &episode.FilePath,
		&episode.FileSize, &episode.Duration, &episode.VideoCodec, &episode.AudioCodec,
		&episode.Resolution, &episode.AudioTracks, &episode.SubtitleTracks,
		&episode.CreatedAt, &episode.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return episode, err
}

// GetEpisodesBySeasonID retrieves all episodes for a season
func (db *DB) GetEpisodesBySeasonID(seasonID int64) ([]*Episode, error) {
	rows, err := db.conn.Query(
		`SELECT id, tv_show_id, season_id, season_number, episode_number, title, overview,
			still_path, air_date, runtime, rating, source_id, file_path, file_size, duration,
			video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM episodes WHERE season_id = ? ORDER BY episode_number`,
		seasonID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEpisodeRows(rows)
}

// GetEpisodesByShowID retrieves all episodes for a TV show
func (db *DB) GetEpisodesByShowID(showID int64) ([]*Episode, error) {
	rows, err := db.conn.Query(
		`SELECT id, tv_show_id, season_id, season_number, episode_number, title, overview,
			still_path, air_date, runtime, rating, source_id, file_path, file_size, duration,
			video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM episodes WHERE tv_show_id = ? ORDER BY season_number, episode_number`,
		showID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEpisodeRows(rows)
}

// GetRandomEpisode retrieves a random episode from a TV show
func (db *DB) GetRandomEpisode(showID int64) (*Episode, error) {
	episode := &Episode{}
	err := db.conn.QueryRow(
		`SELECT id, tv_show_id, season_id, season_number, episode_number, title, overview,
			still_path, air_date, runtime, rating, source_id, file_path, file_size, duration,
			video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM episodes WHERE tv_show_id = ? ORDER BY RANDOM() LIMIT 1`,
		showID,
	).Scan(&episode.ID, &episode.TVShowID, &episode.SeasonID, &episode.SeasonNumber,
		&episode.EpisodeNumber, &episode.Title, &episode.Overview, &episode.StillPath,
		&episode.AirDate, &episode.Runtime, &episode.Rating, &episode.SourceID, &episode.FilePath,
		&episode.FileSize, &episode.Duration, &episode.VideoCodec, &episode.AudioCodec,
		&episode.Resolution, &episode.AudioTracks, &episode.SubtitleTracks,
		&episode.CreatedAt, &episode.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return episode, err
}

// GetRandomEpisodeFromSeason retrieves a random episode from a season
func (db *DB) GetRandomEpisodeFromSeason(seasonID int64) (*Episode, error) {
	episode := &Episode{}
	err := db.conn.QueryRow(
		`SELECT id, tv_show_id, season_id, season_number, episode_number, title, overview,
			still_path, air_date, runtime, rating, source_id, file_path, file_size, duration,
			video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM episodes WHERE season_id = ? ORDER BY RANDOM() LIMIT 1`,
		seasonID,
	).Scan(&episode.ID, &episode.TVShowID, &episode.SeasonID, &episode.SeasonNumber,
		&episode.EpisodeNumber, &episode.Title, &episode.Overview, &episode.StillPath,
		&episode.AirDate, &episode.Runtime, &episode.Rating, &episode.SourceID, &episode.FilePath,
		&episode.FileSize, &episode.Duration, &episode.VideoCodec, &episode.AudioCodec,
		&episode.Resolution, &episode.AudioTracks, &episode.SubtitleTracks,
		&episode.CreatedAt, &episode.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return episode, err
}

func scanEpisodeRows(rows *sql.Rows) ([]*Episode, error) {
	episodes := make([]*Episode, 0)
	for rows.Next() {
		episode := &Episode{}
		if err := rows.Scan(&episode.ID, &episode.TVShowID, &episode.SeasonID, &episode.SeasonNumber,
			&episode.EpisodeNumber, &episode.Title, &episode.Overview, &episode.StillPath,
			&episode.AirDate, &episode.Runtime, &episode.Rating, &episode.SourceID, &episode.FilePath,
			&episode.FileSize, &episode.Duration, &episode.VideoCodec, &episode.AudioCodec,
			&episode.Resolution, &episode.AudioTracks, &episode.SubtitleTracks,
			&episode.CreatedAt, &episode.UpdatedAt); err != nil {
			return nil, err
		}
		episodes = append(episodes, episode)
	}
	return episodes, nil
}

// ============ Extras Repository Methods ============

// CreateExtra creates a new extra content record
func (db *DB) CreateExtra(extra *Extra) (*Extra, error) {
	result, err := db.conn.Exec(
		`INSERT INTO extras (title, category, movie_id, tv_show_id, episode_id, season_number, episode_number,
			source_id, file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		extra.Title, extra.Category, extra.MovieID, extra.TVShowID, extra.EpisodeID,
		extra.SeasonNumber, extra.EpisodeNumber, extra.SourceID, extra.FilePath, extra.FileSize,
		extra.Duration, extra.VideoCodec, extra.AudioCodec, extra.Resolution,
		extra.AudioTracks, extra.SubtitleTracks,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetExtraByID(id)
}

// GetExtraByID retrieves an extra by ID
func (db *DB) GetExtraByID(id int64) (*Extra, error) {
	query := `SELECT id, title, category, movie_id, tv_show_id, episode_id, season_number, episode_number,
		source_id, file_path, file_size, duration, video_codec, audio_codec, resolution,
		audio_tracks, subtitle_tracks, created_at, updated_at
	 FROM extras WHERE id = ?`
	extra, err := getByID(db.conn, query, id, scanExtraRow)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &extra, nil
}

// GetExtraByFilePath checks if an extra with given path exists
func (db *DB) GetExtraByFilePath(filePath string) (*Extra, error) {
	query := `SELECT id, title, category, movie_id, tv_show_id, episode_id, season_number, episode_number,
		source_id, file_path, file_size, duration, video_codec, audio_codec, resolution,
		audio_tracks, subtitle_tracks, created_at, updated_at
	 FROM extras WHERE file_path = ?`
	extra, err := getByFilePath(db.conn, query, filePath, scanExtraRow)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &extra, nil
}

// GetExtrasByMovieID gets all extras for a movie
func (db *DB) GetExtrasByMovieID(movieID int64) ([]*Extra, error) {
	rows, err := db.conn.Query(
		`SELECT id, title, category, movie_id, tv_show_id, episode_id, season_number, episode_number,
			source_id, file_path, file_size, duration, video_codec, audio_codec, resolution,
			audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM extras WHERE movie_id = ? ORDER BY category, title`,
		movieID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExtraRows(rows)
}

// GetExtrasByTVShowID gets all extras for a TV show (including episode-specific)
func (db *DB) GetExtrasByTVShowID(showID int64) ([]*Extra, error) {
	rows, err := db.conn.Query(
		`SELECT e.id, e.title, e.category, e.movie_id, e.tv_show_id, e.episode_id, e.season_number, e.episode_number,
			e.source_id, e.file_path, e.file_size, e.duration, e.video_codec, e.audio_codec, e.resolution,
			e.audio_tracks, e.subtitle_tracks, e.created_at, e.updated_at
		 FROM extras e
		 WHERE e.tv_show_id = ?
		    OR e.episode_id IN (SELECT id FROM episodes WHERE tv_show_id = ?)
		 ORDER BY e.season_number, e.episode_number, e.category, e.title`,
		showID, showID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExtraRows(rows)
}

// GetExtrasByEpisodeID gets episode-specific extras
func (db *DB) GetExtrasByEpisodeID(episodeID int64) ([]*Extra, error) {
	rows, err := db.conn.Query(
		`SELECT id, title, category, movie_id, tv_show_id, episode_id, season_number, episode_number,
			source_id, file_path, file_size, duration, video_codec, audio_codec, resolution,
			audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM extras WHERE episode_id = ? ORDER BY category, title`,
		episodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExtraRows(rows)
}

// GetAllExtras retrieves all extras with pagination
func (db *DB) GetAllExtras(limit, offset int) ([]*Extra, int, error) {
	// Get total count
	var total int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM extras`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.conn.Query(
		`SELECT e.id, e.title, e.category, e.movie_id, e.tv_show_id, e.episode_id, e.season_number, e.episode_number,
			e.source_id, e.file_path, e.file_size, e.duration, e.video_codec, e.audio_codec, e.resolution,
			e.audio_tracks, e.subtitle_tracks, e.created_at, e.updated_at
		 FROM extras e ORDER BY e.created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	extras, err := scanExtraRows(rows)
	return extras, total, err
}

// GetExtrasByCategory gets all extras of a specific category with pagination
func (db *DB) GetExtrasByCategory(category ExtraCategory, limit, offset int) ([]*Extra, int, error) {
	// Get total count
	var total int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM extras WHERE category = ?`, category).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.conn.Query(
		`SELECT id, title, category, movie_id, tv_show_id, episode_id, season_number, episode_number,
			source_id, file_path, file_size, duration, video_codec, audio_codec, resolution,
			audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM extras WHERE category = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		category, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	extras, err := scanExtraRows(rows)
	return extras, total, err
}

// CategoryCount represents a category with its count
type CategoryCount struct {
	Category ExtraCategory `json:"category"`
	Count    int           `json:"count"`
}

// GetExtraCategories returns list of categories with counts
func (db *DB) GetExtraCategories() ([]CategoryCount, error) {
	rows, err := db.conn.Query(
		`SELECT category, COUNT(*) as count FROM extras GROUP BY category ORDER BY count DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]CategoryCount, 0)
	for rows.Next() {
		var cc CategoryCount
		if err := rows.Scan(&cc.Category, &cc.Count); err != nil {
			return nil, err
		}
		categories = append(categories, cc)
	}
	return categories, nil
}

// GetExtrasCount returns total count of extras
func (db *DB) GetExtrasCount() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM extras`).Scan(&count)
	return count, err
}

// GetRandomExtra retrieves a random extra, optionally filtered by category
func (db *DB) GetRandomExtra(category string) (*Extra, error) {
	extra := &Extra{}
	var query string
	var err error

	if category != "" {
		query = `SELECT id, title, category, movie_id, tv_show_id, episode_id, season_number, episode_number,
			source_id, file_path, file_size, duration, video_codec, audio_codec, resolution,
			audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM extras WHERE category = ? ORDER BY RANDOM() LIMIT 1`
		err = db.conn.QueryRow(query, category).Scan(&extra.ID, &extra.Title, &extra.Category,
			&extra.MovieID, &extra.TVShowID, &extra.EpisodeID, &extra.SeasonNumber, &extra.EpisodeNumber,
			&extra.SourceID, &extra.FilePath, &extra.FileSize, &extra.Duration, &extra.VideoCodec,
			&extra.AudioCodec, &extra.Resolution, &extra.AudioTracks, &extra.SubtitleTracks,
			&extra.CreatedAt, &extra.UpdatedAt)
	} else {
		query = `SELECT id, title, category, movie_id, tv_show_id, episode_id, season_number, episode_number,
			source_id, file_path, file_size, duration, video_codec, audio_codec, resolution,
			audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM extras ORDER BY RANDOM() LIMIT 1`
		err = db.conn.QueryRow(query).Scan(&extra.ID, &extra.Title, &extra.Category,
			&extra.MovieID, &extra.TVShowID, &extra.EpisodeID, &extra.SeasonNumber, &extra.EpisodeNumber,
			&extra.SourceID, &extra.FilePath, &extra.FileSize, &extra.Duration, &extra.VideoCodec,
			&extra.AudioCodec, &extra.Resolution, &extra.AudioTracks, &extra.SubtitleTracks,
			&extra.CreatedAt, &extra.UpdatedAt)
	}

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return extra, err
}

// DeleteExtrasBySourceID removes all extras from a source
func (db *DB) DeleteExtrasBySourceID(sourceID int64) error {
	_, err := db.conn.Exec(`DELETE FROM extras WHERE source_id = ?`, sourceID)
	return err
}

func scanExtraRows(rows *sql.Rows) ([]*Extra, error) {
	extras := make([]*Extra, 0)
	for rows.Next() {
		extra := &Extra{}
		if err := rows.Scan(&extra.ID, &extra.Title, &extra.Category, &extra.MovieID, &extra.TVShowID,
			&extra.EpisodeID, &extra.SeasonNumber, &extra.EpisodeNumber, &extra.SourceID, &extra.FilePath,
			&extra.FileSize, &extra.Duration, &extra.VideoCodec, &extra.AudioCodec, &extra.Resolution,
			&extra.AudioTracks, &extra.SubtitleTracks, &extra.CreatedAt, &extra.UpdatedAt); err != nil {
			return nil, err
		}
		extras = append(extras, extra)
	}
	return extras, nil
}

// ==================== Section Methods ====================

// GetAllSections returns all sections ordered by display_order
func (db *DB) GetAllSections() ([]Section, error) {
	query := `
        SELECT id, name, slug, COALESCE(icon, ''), COALESCE(description, ''), section_type,
               display_order, is_visible, created_at, updated_at
        FROM sections
        ORDER BY display_order ASC
    `

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sections []Section
	for rows.Next() {
		var s Section
		err := rows.Scan(
			&s.ID, &s.Name, &s.Slug, &s.Icon, &s.Description,
			&s.SectionType, &s.DisplayOrder, &s.IsVisible,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sections = append(sections, s)
	}

	return sections, rows.Err()
}

// GetVisibleSections returns only visible sections
func (db *DB) GetVisibleSections() ([]Section, error) {
	query := `
        SELECT id, name, slug, COALESCE(icon, ''), COALESCE(description, ''), section_type,
               display_order, is_visible, created_at, updated_at
        FROM sections
        WHERE is_visible = 1
        ORDER BY display_order ASC
    `

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sections []Section
	for rows.Next() {
		var s Section
		err := rows.Scan(
			&s.ID, &s.Name, &s.Slug, &s.Icon, &s.Description,
			&s.SectionType, &s.DisplayOrder, &s.IsVisible,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sections = append(sections, s)
	}

	return sections, rows.Err()
}

// GetSectionByID retrieves a section by ID
func (db *DB) GetSectionByID(id int64) (*Section, error) {
	query := `
        SELECT id, name, slug, COALESCE(icon, ''), COALESCE(description, ''), section_type,
               display_order, is_visible, created_at, updated_at
        FROM sections
        WHERE id = ?
    `

	var s Section
	err := db.conn.QueryRow(query, id).Scan(
		&s.ID, &s.Name, &s.Slug, &s.Icon, &s.Description,
		&s.SectionType, &s.DisplayOrder, &s.IsVisible,
		&s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// GetSectionBySlug retrieves a section by slug
func (db *DB) GetSectionBySlug(slug string) (*Section, error) {
	query := `
        SELECT id, name, slug, COALESCE(icon, ''), COALESCE(description, ''), section_type,
               display_order, is_visible, created_at, updated_at
        FROM sections
        WHERE slug = ?
    `

	var s Section
	err := db.conn.QueryRow(query, slug).Scan(
		&s.ID, &s.Name, &s.Slug, &s.Icon, &s.Description,
		&s.SectionType, &s.DisplayOrder, &s.IsVisible,
		&s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// CreateSection creates a new section
func (db *DB) CreateSection(section *Section) error {
	query := `
        INSERT INTO sections (name, slug, icon, description, section_type, display_order, is_visible)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `

	result, err := db.conn.Exec(query,
		section.Name, section.Slug, section.Icon, section.Description,
		section.SectionType, section.DisplayOrder, section.IsVisible,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	section.ID = id
	return nil
}

// UpdateSection updates an existing section
func (db *DB) UpdateSection(section *Section) error {
	query := `
        UPDATE sections
        SET name = ?, slug = ?, icon = ?, description = ?,
            section_type = ?, display_order = ?, is_visible = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `

	_, err := db.conn.Exec(query,
		section.Name, section.Slug, section.Icon, section.Description,
		section.SectionType, section.DisplayOrder, section.IsVisible,
		section.ID,
	)

	return err
}

// DeleteSection deletes a section
func (db *DB) DeleteSection(id int64) error {
	query := `DELETE FROM sections WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// GetSectionRules returns all rules for a section
func (db *DB) GetSectionRules(sectionID int64) ([]SectionRule, error) {
	query := `
        SELECT id, section_id, field, operator, value, created_at
        FROM section_rules
        WHERE section_id = ?
        ORDER BY id ASC
    `

	rows, err := db.conn.Query(query, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []SectionRule
	for rows.Next() {
		var r SectionRule
		err := rows.Scan(&r.ID, &r.SectionID, &r.Field, &r.Operator, &r.Value, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	return rules, rows.Err()
}

// CreateSectionRule creates a new rule for a section
func (db *DB) CreateSectionRule(rule *SectionRule) error {
	query := `
        INSERT INTO section_rules (section_id, field, operator, value)
        VALUES (?, ?, ?, ?)
    `

	result, err := db.conn.Exec(query, rule.SectionID, rule.Field, rule.Operator, rule.Value)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	rule.ID = id
	return nil
}

// DeleteSectionRule deletes a rule
func (db *DB) DeleteSectionRule(id int64) error {
	query := `DELETE FROM section_rules WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// AddMediaToSection adds a media item to a section
func (db *DB) AddMediaToSection(mediaID int64, mediaType MediaType, sectionID int64) error {
	query := `
        INSERT OR IGNORE INTO media_sections (media_id, media_type, section_id)
        VALUES (?, ?, ?)
    `

	_, err := db.conn.Exec(query, mediaID, mediaType, sectionID)
	return err
}

// RemoveMediaFromSection removes a media item from a section
func (db *DB) RemoveMediaFromSection(mediaID int64, mediaType MediaType, sectionID int64) error {
	query := `
        DELETE FROM media_sections
        WHERE media_id = ? AND media_type = ? AND section_id = ?
    `

	_, err := db.conn.Exec(query, mediaID, mediaType, sectionID)
	return err
}

// GetMediaSections returns all section IDs a media item belongs to
func (db *DB) GetMediaSections(mediaID int64, mediaType MediaType) ([]int64, error) {
	query := `
        SELECT section_id
        FROM media_sections
        WHERE media_id = ? AND media_type = ?
    `

	rows, err := db.conn.Query(query, mediaID, mediaType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sectionIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		sectionIDs = append(sectionIDs, id)
	}

	return sectionIDs, rows.Err()
}

// GetMediaBySectionID returns media items in a section
func (db *DB) GetMediaBySectionID(sectionID int64, limit, offset int) ([]interface{}, int, error) {
	// First get the section to determine its type
	section, err := db.GetSectionByID(sectionID)
	if err != nil {
		return nil, 0, err
	}

	var items []interface{}
	var total int

	if section.SectionType == SectionTypeSmart {
		// For smart sections, evaluate rules
		items, total, err = db.evaluateSmartSection(section, limit, offset)
	} else {
		// For standard sections, get manually assigned media
		items, total, err = db.getManualSectionMedia(sectionID, limit, offset)
	}

	return items, total, err
}

// Helper method for manual sections
func (db *DB) getManualSectionMedia(sectionID int64, limit, offset int) ([]interface{}, int, error) {
	// Get total count
	countQuery := `
        SELECT COUNT(*)
        FROM media_sections
        WHERE section_id = ?
    `

	var total int
	err := db.conn.QueryRow(countQuery, sectionID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get media items
	query := `
        SELECT ms.media_id, ms.media_type
        FROM media_sections ms
        WHERE ms.section_id = ?
        ORDER BY ms.added_at DESC
        LIMIT ? OFFSET ?
    `

	rows, err := db.conn.Query(query, sectionID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []interface{}
	for rows.Next() {
		var mediaID int64
		var mediaType MediaType

		if err := rows.Scan(&mediaID, &mediaType); err != nil {
			continue
		}

		// Fetch the actual media item based on type
		switch mediaType {
		case MediaTypeMovie:
			if media, err := db.GetMediaByID(mediaID); err == nil {
				items = append(items, media)
			}
		case MediaTypeEpisode:
			if episode, err := db.GetEpisodeByID(mediaID); err == nil {
				items = append(items, episode)
			}
		case MediaTypeExtra:
			if extra, err := db.GetExtraByID(mediaID); err == nil {
				items = append(items, extra)
			}
		}
	}

	return items, total, nil
}

// ============ Library Statistics ============

// LibraryStats contains aggregate library statistics
type LibraryStats struct {
	MovieCount   int `json:"movie_count"`
	ShowCount    int `json:"show_count"`
	EpisodeCount int `json:"episode_count"`
	ExtraCount   int `json:"extra_count"`
	SourceCount  int `json:"source_count"`
}

// GetLibraryStats returns aggregate statistics for the media library
func (db *DB) GetLibraryStats() (*LibraryStats, error) {
	stats := &LibraryStats{}

	query := `
		SELECT
			(SELECT COUNT(*) FROM media WHERE type = 'movie') as movies,
			(SELECT COUNT(*) FROM tv_shows) as shows,
			(SELECT COUNT(*) FROM episodes) as episodes,
			(SELECT COUNT(*) FROM extras) as extras,
			(SELECT COUNT(*) FROM media_sources WHERE enabled = 1) as sources
	`

	err := db.conn.QueryRow(query).Scan(
		&stats.MovieCount, &stats.ShowCount, &stats.EpisodeCount,
		&stats.ExtraCount, &stats.SourceCount,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// ============ Channel Repository Methods ============

// CreateChannel creates a new channel for a user
func (db *DB) CreateChannel(userID int64, name, description, icon string) (*Channel, error) {
	if icon == "" {
		icon = "📺"
	}

	result, err := db.conn.Exec(
		`INSERT INTO channels (user_id, name, description, icon) VALUES (?, ?, ?, ?)`,
		userID, name, description, icon,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetChannelByID(id)
}

// GetChannelByID retrieves a channel by ID
func (db *DB) GetChannelByID(id int64) (*Channel, error) {
	channel := &Channel{}
	err := db.conn.QueryRow(
		`SELECT id, user_id, name, description, icon, created_at, updated_at FROM channels WHERE id = ?`,
		id,
	).Scan(&channel.ID, &channel.UserID, &channel.Name, &channel.Description, &channel.Icon, &channel.CreatedAt, &channel.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Load sources
	sources, err := db.GetChannelSources(id)
	if err == nil {
		channel.Sources = sources
	}

	return channel, nil
}

// GetUserChannels retrieves all channels for a user
func (db *DB) GetUserChannels(userID int64) ([]Channel, error) {
	// Use a single query with LEFT JOIN to get channel data and schedule stats together
	// This avoids nested queries which can cause SQLite deadlocks
	rows, err := db.conn.Query(
		`SELECT c.id, c.user_id, c.name, c.description, c.icon, c.created_at, c.updated_at,
			COALESCE(s.item_count, 0) as item_count,
			COALESCE(s.total_duration, 0) as total_duration
		FROM channels c
		LEFT JOIN (
			SELECT channel_id, COUNT(*) as item_count, SUM(duration) as total_duration
			FROM channel_schedule
			WHERE cycle_number = 1
			GROUP BY channel_id
		) s ON c.id = s.channel_id
		WHERE c.user_id = ?
		ORDER BY c.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []Channel
	for rows.Next() {
		var ch Channel
		if err := rows.Scan(&ch.ID, &ch.UserID, &ch.Name, &ch.Description, &ch.Icon, &ch.CreatedAt, &ch.UpdatedAt, &ch.ItemCount, &ch.TotalDuration); err != nil {
			continue
		}
		channels = append(channels, ch)
	}

	return channels, rows.Err()
}

// UpdateChannel updates a channel's details
func (db *DB) UpdateChannel(id int64, name, description, icon string) (*Channel, error) {
	_, err := db.conn.Exec(
		`UPDATE channels SET name = ?, description = ?, icon = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		name, description, icon, id,
	)
	if err != nil {
		return nil, err
	}
	return db.GetChannelByID(id)
}

// DeleteChannel deletes a channel and all its data
func (db *DB) DeleteChannel(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM channels WHERE id = ?`, id)
	return err
}

// ============ Channel Sources ============

// AddChannelSource adds a content source to a channel
func (db *DB) AddChannelSource(channelID int64, sourceType string, sourceID *int64, sourceValue string, weight int, shuffle bool, options *ChannelSourceOptions) (*ChannelSource, error) {
	if weight < 1 {
		weight = 1
	}

	var optionsJSON *string
	if options != nil {
		data, err := json.Marshal(options)
		if err != nil {
			return nil, err
		}
		s := string(data)
		optionsJSON = &s
	}

	result, err := db.conn.Exec(
		`INSERT INTO channel_sources (channel_id, source_type, source_id, source_value, weight, shuffle, options) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		channelID, sourceType, sourceID, sourceValue, weight, shuffle, optionsJSON,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return db.GetChannelSourceByID(id)
}

// GetChannelSourceByID retrieves a channel source by ID
func (db *DB) GetChannelSourceByID(id int64) (*ChannelSource, error) {
	source := &ChannelSource{}
	var optionsJSON sql.NullString
	err := db.conn.QueryRow(
		`SELECT id, channel_id, source_type, source_id, source_value, weight, (shuffle > 0) as shuffle, options FROM channel_sources WHERE id = ?`,
		id,
	).Scan(&source.ID, &source.ChannelID, &source.SourceType, &source.SourceID, &source.SourceValue, &source.Weight, &source.Shuffle, &optionsJSON)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if optionsJSON.Valid && optionsJSON.String != "" {
		var opts ChannelSourceOptions
		if json.Unmarshal([]byte(optionsJSON.String), &opts) == nil {
			source.Options = &opts
		}
	}

	return source, nil
}

// GetChannelSources retrieves all sources for a channel
func (db *DB) GetChannelSources(channelID int64) ([]ChannelSource, error) {
	rows, err := db.conn.Query(
		`SELECT id, channel_id, source_type, source_id, source_value, weight, (shuffle > 0) as shuffle, options FROM channel_sources WHERE channel_id = ?`,
		channelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// First pass: collect all sources without nested queries
	var sources []ChannelSource
	for rows.Next() {
		var s ChannelSource
		var optionsJSON sql.NullString
		if err := rows.Scan(&s.ID, &s.ChannelID, &s.SourceType, &s.SourceID, &s.SourceValue, &s.Weight, &s.Shuffle, &optionsJSON); err != nil {
			continue
		}
		if optionsJSON.Valid && optionsJSON.String != "" {
			var opts ChannelSourceOptions
			if json.Unmarshal([]byte(optionsJSON.String), &opts) == nil {
				s.Options = &opts
			}
		}
		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Second pass: populate source names (now safe since rows is closed)
	for i := range sources {
		sources[i].SourceName = db.getSourceName(sources[i].SourceType, sources[i].SourceID, sources[i].SourceValue)
	}

	return sources, nil
}

// getSourceName returns a display name for a channel source
func (db *DB) getSourceName(sourceType string, sourceID *int64, sourceValue string) string {
	if sourceID == nil {
		if sourceValue != "" {
			return sourceValue
		}
		return "Unknown"
	}

	var name string
	switch sourceType {
	case ChannelSourcePlaylist:
		db.conn.QueryRow(`SELECT name FROM playlists WHERE id = ?`, *sourceID).Scan(&name)
	case ChannelSourceSection:
		db.conn.QueryRow(`SELECT name FROM sections WHERE id = ?`, *sourceID).Scan(&name)
	case ChannelSourceShow:
		db.conn.QueryRow(`SELECT title FROM tv_shows WHERE id = ?`, *sourceID).Scan(&name)
	case ChannelSourceMovie:
		db.conn.QueryRow(`SELECT title FROM media WHERE id = ?`, *sourceID).Scan(&name)
	}

	if name == "" {
		return "Unknown"
	}
	return name
}

// DeleteChannelSource removes a source from a channel
func (db *DB) DeleteChannelSource(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM channel_sources WHERE id = ?`, id)
	return err
}

// ============ Channel Schedule Generation ============

// channelScheduleInput is used internally for schedule generation
type channelScheduleInput struct {
	MediaID   int64
	MediaType MediaType
	Duration  int
	Title     string
}

// GenerateChannelSchedule generates or regenerates a channel's schedule
func (db *DB) GenerateChannelSchedule(channelID int64) error {
	// Seed random number generator
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Get all sources for this channel
	sources, err := db.GetChannelSources(channelID)
	if err != nil {
		return err
	}

	if len(sources) == 0 {
		return nil // No sources, nothing to schedule
	}

	// Build per-source item lists
	type sourceWithItems struct {
		source ChannelSource
		items  []channelScheduleInput
	}
	var sourcesWithItems []sourceWithItems

	for _, source := range sources {
		items := db.getMediaFromSource(source)
		if len(items) == 0 {
			continue // Skip empty sources
		}

		// Shuffle items only if source.Shuffle is true
		if source.Shuffle {
			rng.Shuffle(len(items), func(i, j int) {
				items[i], items[j] = items[j], items[i]
			})
		}

		sourcesWithItems = append(sourcesWithItems, sourceWithItems{source: source, items: items})
	}

	if len(sourcesWithItems) == 0 {
		return nil
	}

	var finalItems []channelScheduleInput

	// Check if all sources have equal weight (fair rotation mode)
	allEqualWeight := true
	firstWeight := sourcesWithItems[0].source.Weight
	for _, sw := range sourcesWithItems {
		if sw.source.Weight != firstWeight {
			allEqualWeight = false
			break
		}
	}

	if allEqualWeight && len(sourcesWithItems) > 1 {
		// Fair rotation: cycle through sources in round-robin fashion
		// Each source gets weight repetitions per cycle
		weight := firstWeight

		// Find the maximum number of items across all sources
		maxItems := 0
		for _, sw := range sourcesWithItems {
			if len(sw.items) > maxItems {
				maxItems = len(sw.items)
			}
		}

		// Repeat for weight * maxItems rounds to ensure full coverage
		totalRounds := weight * maxItems

		// Build source order indices
		sourceOrder := make([]int, len(sourcesWithItems))
		for i := range sourceOrder {
			sourceOrder[i] = i
		}

		// Shuffle source order ONCE at the start for variety
		rng.Shuffle(len(sourceOrder), func(i, j int) {
			sourceOrder[i], sourceOrder[j] = sourceOrder[j], sourceOrder[i]
		})

		// Round-robin through sources with consistent order
		for round := 0; round < totalRounds; round++ {
			for _, srcIdx := range sourceOrder {
				sw := sourcesWithItems[srcIdx]
				itemIdx := round % len(sw.items)
				finalItems = append(finalItems, sw.items[itemIdx])
			}
		}
	} else {
		// Weighted shuffle: multiply items by weight and shuffle together
		for _, sw := range sourcesWithItems {
			for i := 0; i < sw.source.Weight; i++ {
				finalItems = append(finalItems, sw.items...)
			}
		}

		// Shuffle the combined list
		rng.Shuffle(len(finalItems), func(i, j int) {
			finalItems[i], finalItems[j] = finalItems[j], finalItems[i]
		})
	}

	if len(finalItems) == 0 {
		return nil
	}

	// Clear existing schedule
	_, err = db.conn.Exec(`DELETE FROM channel_schedule WHERE channel_id = ?`, channelID)
	if err != nil {
		return err
	}

	// Insert new schedule with cumulative timing
	cumulativeStart := 0
	for position, item := range finalItems {
		_, err = db.conn.Exec(
			`INSERT INTO channel_schedule (channel_id, media_id, media_type, scheduled_position, cycle_number, duration, cumulative_start)
			VALUES (?, ?, ?, ?, 1, ?, ?)`,
			channelID, item.MediaID, item.MediaType, position, item.Duration, cumulativeStart,
		)
		if err != nil {
			return err
		}
		cumulativeStart += item.Duration
	}

	return nil
}

// getMediaFromSource extracts media items from a channel source
func (db *DB) getMediaFromSource(source ChannelSource) []channelScheduleInput {
	var items []channelScheduleInput

	switch source.SourceType {
	case ChannelSourceShow:
		if source.SourceID != nil {
			// Build query with optional season filtering
			query := `SELECT id, title, duration, season_number FROM episodes WHERE tv_show_id = ? AND duration > 0`
			rows, err := db.conn.Query(query, *source.SourceID)
			if err == nil {
				defer rows.Close()

				// Build season filter set if options specify seasons
				var seasonFilter map[int]bool
				if source.Options != nil && len(source.Options.Seasons) > 0 {
					seasonFilter = make(map[int]bool)
					for _, s := range source.Options.Seasons {
						seasonFilter[s] = true
					}
				}

				for rows.Next() {
					var i channelScheduleInput
					var duration sql.NullInt64
					var seasonNum int
					if rows.Scan(&i.MediaID, &i.Title, &duration, &seasonNum) == nil {
						// Skip if season filtering is active and this season isn't selected
						if seasonFilter != nil && !seasonFilter[seasonNum] {
							continue
						}
						i.MediaType = MediaTypeEpisode
						i.Duration = int(duration.Int64)
						if i.Duration > 0 {
							items = append(items, i)
						}
					}
				}
			}

			// Add commentary extras if requested
			if source.Options != nil && source.Options.IncludeCommentary {
				extras := db.getShowExtrasForChannel(*source.SourceID, []string{string(ExtraCategoryCommentary)}, nil)
				items = append(items, extras...)
			}

			// Add other extras by category if requested
			if source.Options != nil && len(source.Options.ExtrasCategories) > 0 {
				extras := db.getShowExtrasForChannel(*source.SourceID, source.Options.ExtrasCategories, nil)
				items = append(items, extras...)
			}
		}

	case ChannelSourceMovie:
		if source.SourceID != nil {
			var i channelScheduleInput
			var duration sql.NullInt64
			err := db.conn.QueryRow(
				`SELECT id, title, duration FROM media WHERE id = ? AND type = 'movie'`,
				*source.SourceID,
			).Scan(&i.MediaID, &i.Title, &duration)
			if err == nil {
				i.MediaType = MediaTypeMovie
				i.Duration = int(duration.Int64)
				if i.Duration > 0 {
					items = append(items, i)
				}
			}
		}

	case ChannelSourcePlaylist:
		if source.SourceID != nil {
			rows, err := db.conn.Query(
				`SELECT pi.media_id, pi.media_type,
					CASE
						WHEN pi.media_type = 'movie' THEN (SELECT duration FROM media WHERE id = pi.media_id)
						WHEN pi.media_type = 'episode' THEN (SELECT duration FROM episodes WHERE id = pi.media_id)
						WHEN pi.media_type = 'extra' THEN (SELECT duration FROM extras WHERE id = pi.media_id)
					END as duration,
					CASE
						WHEN pi.media_type = 'movie' THEN (SELECT title FROM media WHERE id = pi.media_id)
						WHEN pi.media_type = 'episode' THEN (SELECT title FROM episodes WHERE id = pi.media_id)
						WHEN pi.media_type = 'extra' THEN (SELECT title FROM extras WHERE id = pi.media_id)
					END as title
				FROM playlist_items pi
				WHERE pi.playlist_id = ?
				ORDER BY pi.position`,
				*source.SourceID,
			)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var i channelScheduleInput
					var duration sql.NullInt64
					var title sql.NullString
					if rows.Scan(&i.MediaID, &i.MediaType, &duration, &title) == nil {
						i.Duration = int(duration.Int64)
						i.Title = title.String
						if i.Duration > 0 {
							items = append(items, i)
						}
					}
				}
			}
		}

	case ChannelSourceExtraCategory:
		rows, err := db.conn.Query(
			`SELECT id, title, duration FROM extras WHERE category = ? AND duration > 0`,
			source.SourceValue,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var i channelScheduleInput
				var duration sql.NullInt64
				if rows.Scan(&i.MediaID, &i.Title, &duration) == nil {
					i.MediaType = MediaTypeExtra
					i.Duration = int(duration.Int64)
					if i.Duration > 0 {
						items = append(items, i)
					}
				}
			}
		}
	}

	return items
}

// getShowExtrasForChannel retrieves extras for a TV show filtered by categories
func (db *DB) getShowExtrasForChannel(showID int64, categories []string, seasons []int) []channelScheduleInput {
	var items []channelScheduleInput

	if len(categories) == 0 {
		return items
	}

	// Build the query with category filter
	query := `SELECT id, title, duration, category, season_number FROM extras WHERE tv_show_id = ? AND duration > 0 AND category IN (`
	args := []interface{}{showID}
	for i, cat := range categories {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, cat)
	}
	query += ")"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return items
	}
	defer rows.Close()

	// Build season filter if provided
	var seasonFilter map[int]bool
	if len(seasons) > 0 {
		seasonFilter = make(map[int]bool)
		for _, s := range seasons {
			seasonFilter[s] = true
		}
	}

	for rows.Next() {
		var i channelScheduleInput
		var duration sql.NullInt64
		var category string
		var seasonNum sql.NullInt64
		if rows.Scan(&i.MediaID, &i.Title, &duration, &category, &seasonNum) == nil {
			// Skip if season filtering is active and this extra has a season that isn't selected
			if seasonFilter != nil && seasonNum.Valid && !seasonFilter[int(seasonNum.Int64)] {
				continue
			}
			i.MediaType = MediaTypeExtra
			i.Duration = int(duration.Int64)
			if i.Duration > 0 {
				items = append(items, i)
			}
		}
	}

	return items
}

// ShowOptionsInfo contains available options for a TV show source
type ShowOptionsInfo struct {
	Seasons          []SeasonInfo         `json:"seasons"`
	HasCommentary    bool                 `json:"has_commentary"`
	ExtrasCategories []ExtraCategoryInfo  `json:"extras_categories"`
}

// SeasonInfo contains season number and episode count
type SeasonInfo struct {
	Number       int `json:"number"`
	EpisodeCount int `json:"episode_count"`
}

// ExtraCategoryInfo contains extras category name and count
type ExtraCategoryInfo struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// GetShowOptionsForChannel retrieves available options for a TV show source
func (db *DB) GetShowOptionsForChannel(showID int64) (*ShowOptionsInfo, error) {
	info := &ShowOptionsInfo{
		Seasons:          []SeasonInfo{},
		ExtrasCategories: []ExtraCategoryInfo{},
	}

	// Get seasons with episode counts
	rows, err := db.conn.Query(
		`SELECT season_number, COUNT(*) as count FROM episodes
		 WHERE tv_show_id = ? AND duration > 0
		 GROUP BY season_number ORDER BY season_number`,
		showID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s SeasonInfo
		if rows.Scan(&s.Number, &s.EpisodeCount) == nil {
			info.Seasons = append(info.Seasons, s)
		}
	}

	// Check for commentary extras
	var commentaryCount int
	db.conn.QueryRow(
		`SELECT COUNT(*) FROM extras WHERE tv_show_id = ? AND category = 'commentary' AND duration > 0`,
		showID,
	).Scan(&commentaryCount)
	info.HasCommentary = commentaryCount > 0

	// Get extras categories with counts
	extrasRows, err := db.conn.Query(
		`SELECT category, COUNT(*) as count FROM extras
		 WHERE tv_show_id = ? AND category != 'commentary' AND duration > 0
		 GROUP BY category ORDER BY category`,
		showID,
	)
	if err == nil {
		defer extrasRows.Close()
		for extrasRows.Next() {
			var e ExtraCategoryInfo
			if extrasRows.Scan(&e.Category, &e.Count) == nil {
				info.ExtrasCategories = append(info.ExtrasCategories, e)
			}
		}
	}

	return info, nil
}

// ============ Channel Now Playing ============

// GetChannelNowPlaying calculates what's currently playing on a channel
func (db *DB) GetChannelNowPlaying(channelID int64) (*ChannelNowPlaying, error) {
	channel, err := db.GetChannelByID(channelID)
	if err != nil {
		return nil, err
	}

	// Get total cycle duration
	var totalDuration sql.NullInt64
	err = db.conn.QueryRow(
		`SELECT COALESCE(SUM(duration), 0) FROM channel_schedule WHERE channel_id = ? AND cycle_number = 1`,
		channelID,
	).Scan(&totalDuration)
	if err != nil || totalDuration.Int64 == 0 {
		return &ChannelNowPlaying{Channel: *channel}, nil
	}

	cycleDuration := int(totalDuration.Int64)

	// Calculate current position in cycle based on time
	// Use channel creation time as epoch for consistent scheduling
	elapsed := int(time.Since(channel.CreatedAt).Seconds())
	positionInCycle := elapsed % cycleDuration

	// Find current item
	var current ChannelScheduleItem
	err = db.conn.QueryRow(
		`SELECT cs.id, cs.channel_id, cs.media_id, cs.media_type, cs.scheduled_position,
			cs.cycle_number, cs.duration, cs.cumulative_start, cs.played
		FROM channel_schedule cs
		WHERE cs.channel_id = ? AND cs.cycle_number = 1
			AND cs.cumulative_start <= ?
			AND cs.cumulative_start + cs.duration > ?
		LIMIT 1`,
		channelID, positionInCycle, positionInCycle,
	).Scan(
		&current.ID, &current.ChannelID, &current.MediaID, &current.MediaType,
		&current.ScheduledPosition, &current.CycleNumber, &current.Duration,
		&current.CumulativeStart, &current.Played,
	)
	if err != nil {
		return &ChannelNowPlaying{Channel: *channel}, nil
	}

	// Populate title and poster for current item
	db.populateScheduleItemDetails(&current)

	// Calculate elapsed time within current item
	elapsedInItem := positionInCycle - current.CumulativeStart

	// Get up next items (next 3)
	var upNext []ChannelScheduleItem
	rows, err := db.conn.Query(
		`SELECT cs.id, cs.channel_id, cs.media_id, cs.media_type, cs.scheduled_position,
			cs.cycle_number, cs.duration, cs.cumulative_start, cs.played
		FROM channel_schedule cs
		WHERE cs.channel_id = ? AND cs.cycle_number = 1
			AND cs.scheduled_position > ?
		ORDER BY cs.scheduled_position
		LIMIT 3`,
		channelID, current.ScheduledPosition,
	)
	if err == nil {
		// First pass: collect items without nested queries
		for rows.Next() {
			var item ChannelScheduleItem
			if rows.Scan(
				&item.ID, &item.ChannelID, &item.MediaID, &item.MediaType,
				&item.ScheduledPosition, &item.CycleNumber, &item.Duration,
				&item.CumulativeStart, &item.Played,
			) == nil {
				upNext = append(upNext, item)
			}
		}
		rows.Close() // Close immediately before any nested queries
	}

	// Second pass: populate details (safe now that rows is closed)
	for i := range upNext {
		db.populateScheduleItemDetails(&upNext[i])
	}

	return &ChannelNowPlaying{
		Channel:    *channel,
		NowPlaying: &current,
		Elapsed:    elapsedInItem,
		UpNext:     upNext,
		CycleStart: channel.CreatedAt,
	}, nil
}

// populateScheduleItemDetails fills in title and poster for a schedule item
func (db *DB) populateScheduleItemDetails(item *ChannelScheduleItem) {
	switch item.MediaType {
	case MediaTypeMovie:
		db.conn.QueryRow(
			`SELECT title, poster_path, backdrop_path FROM media WHERE id = ?`,
			item.MediaID,
		).Scan(&item.Title, &item.PosterPath, &item.BackdropPath)
	case MediaTypeEpisode:
		db.conn.QueryRow(
			`SELECT e.title, t.poster_path, t.backdrop_path
			FROM episodes e
			JOIN tv_shows t ON e.tv_show_id = t.id
			WHERE e.id = ?`,
			item.MediaID,
		).Scan(&item.Title, &item.PosterPath, &item.BackdropPath)
	case MediaTypeExtra:
		db.conn.QueryRow(
			`SELECT title FROM extras WHERE id = ?`,
			item.MediaID,
		).Scan(&item.Title)
	}
}

// GetChannelSchedule returns the full schedule for a channel
func (db *DB) GetChannelSchedule(channelID int64, limit, offset int) ([]ChannelScheduleItem, int, error) {
	// Get total count
	var total int
	db.conn.QueryRow(
		`SELECT COUNT(*) FROM channel_schedule WHERE channel_id = ? AND cycle_number = 1`,
		channelID,
	).Scan(&total)

	// Get items
	rows, err := db.conn.Query(
		`SELECT id, channel_id, media_id, media_type, scheduled_position,
			cycle_number, duration, cumulative_start, played
		FROM channel_schedule
		WHERE channel_id = ? AND cycle_number = 1
		ORDER BY scheduled_position
		LIMIT ? OFFSET ?`,
		channelID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}

	// First pass: collect items without nested queries
	var items []ChannelScheduleItem
	for rows.Next() {
		var item ChannelScheduleItem
		if rows.Scan(
			&item.ID, &item.ChannelID, &item.MediaID, &item.MediaType,
			&item.ScheduledPosition, &item.CycleNumber, &item.Duration,
			&item.CumulativeStart, &item.Played,
		) == nil {
			items = append(items, item)
		}
	}
	rowsErr := rows.Err()
	rows.Close() // Close before nested queries

	// Second pass: populate details
	for i := range items {
		db.populateScheduleItemDetails(&items[i])
	}

	return items, total, rowsErr
}
