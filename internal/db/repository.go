package db

import (
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("record not found")

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
	media := &Media{}
	err := db.conn.QueryRow(
		`SELECT id, title, original_title, type, year, overview, poster_path, backdrop_path,
			rating, runtime, genres, tmdb_id, imdb_id, season_count, episode_count, source_id,
			file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks,
			subtitle_tracks, created_at, updated_at
		 FROM media WHERE id = ?`,
		id,
	).Scan(&media.ID, &media.Title, &media.OriginalTitle, &media.Type, &media.Year,
		&media.Overview, &media.PosterPath, &media.BackdropPath, &media.Rating, &media.Runtime,
		&media.Genres, &media.TMDbID, &media.IMDbID, &media.SeasonCount, &media.EpisodeCount,
		&media.SourceID, &media.FilePath, &media.FileSize, &media.Duration, &media.VideoCodec,
		&media.AudioCodec, &media.Resolution, &media.AudioTracks, &media.SubtitleTracks,
		&media.CreatedAt, &media.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return media, err
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
	media := &Media{}
	err := db.conn.QueryRow(
		`SELECT id, title, original_title, type, year, overview, poster_path, backdrop_path,
			rating, runtime, genres, tmdb_id, imdb_id, season_count, episode_count, source_id,
			file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks,
			subtitle_tracks, created_at, updated_at
		 FROM media WHERE file_path = ?`,
		filePath,
	).Scan(&media.ID, &media.Title, &media.OriginalTitle, &media.Type, &media.Year,
		&media.Overview, &media.PosterPath, &media.BackdropPath, &media.Rating, &media.Runtime,
		&media.Genres, &media.TMDbID, &media.IMDbID, &media.SeasonCount, &media.EpisodeCount,
		&media.SourceID, &media.FilePath, &media.FileSize, &media.Duration, &media.VideoCodec,
		&media.AudioCodec, &media.Resolution, &media.AudioTracks, &media.SubtitleTracks,
		&media.CreatedAt, &media.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return media, err
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
		`SELECT p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM playlist_items WHERE playlist_id = p.id) as item_count
		 FROM playlists p WHERE p.id = ?`,
		id,
	).Scan(&playlist.ID, &playlist.UserID, &playlist.Name, &playlist.Description,
		&playlist.CreatedAt, &playlist.UpdatedAt, &playlist.ItemCount)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return playlist, err
}

// GetUserPlaylists retrieves all playlists for a user
func (db *DB) GetUserPlaylists(userID int64) ([]*Playlist, error) {
	rows, err := db.conn.Query(
		`SELECT p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM playlist_items WHERE playlist_id = p.id) as item_count
		 FROM playlists p WHERE p.user_id = ? ORDER BY p.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	playlists := make([]*Playlist, 0)
	for rows.Next() {
		p := &Playlist{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description,
			&p.CreatedAt, &p.UpdatedAt, &p.ItemCount); err != nil {
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
	rows, err := db.conn.Query(
		`SELECT pi.id, pi.playlist_id, pi.media_id, pi.media_type, pi.position, pi.added_at,
			m.title, m.year, m.poster_path, m.duration, m.overview, m.rating, m.resolution
		 FROM playlist_items pi
		 JOIN media m ON pi.media_id = m.id
		 WHERE pi.playlist_id = ?
		 ORDER BY pi.position ASC`,
		playlistID,
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
