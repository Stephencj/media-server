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

// GetTVShowByID retrieves a TV show by ID
func (db *DB) GetTVShowByID(id int64) (*TVShow, error) {
	show := &TVShow{}
	err := db.conn.QueryRow(
		`SELECT id, title, original_title, year, overview, poster_path, backdrop_path,
			rating, genres, tmdb_id, imdb_id, status, created_at, updated_at
		 FROM tv_shows WHERE id = ?`,
		id,
	).Scan(&show.ID, &show.Title, &show.OriginalTitle, &show.Year, &show.Overview,
		&show.PosterPath, &show.BackdropPath, &show.Rating, &show.Genres, &show.TMDbID,
		&show.IMDbID, &show.Status, &show.CreatedAt, &show.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return show, err
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

// GetAllTVShows retrieves all TV shows with season/episode counts
func (db *DB) GetAllTVShows(limit, offset int) ([]*TVShow, int, error) {
	// Get total count
	var total int
	db.conn.QueryRow(`SELECT COUNT(*) FROM tv_shows`).Scan(&total)

	rows, err := db.conn.Query(
		`SELECT s.id, s.title, s.original_title, s.year, s.overview, s.poster_path, s.backdrop_path,
			s.rating, s.genres, s.tmdb_id, s.imdb_id, s.status, s.created_at, s.updated_at,
			(SELECT COUNT(*) FROM seasons WHERE tv_show_id = s.id) as season_count,
			(SELECT COUNT(*) FROM episodes WHERE tv_show_id = s.id) as episode_count
		 FROM tv_shows s ORDER BY s.title LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	shows := make([]*TVShow, 0)
	for rows.Next() {
		show := &TVShow{}
		var seasonCount, episodeCount int
		if err := rows.Scan(&show.ID, &show.Title, &show.OriginalTitle, &show.Year, &show.Overview,
			&show.PosterPath, &show.BackdropPath, &show.Rating, &show.Genres, &show.TMDbID,
			&show.IMDbID, &show.Status, &show.CreatedAt, &show.UpdatedAt,
			&seasonCount, &episodeCount); err != nil {
			return nil, 0, err
		}
		show.SeasonCount = seasonCount
		show.EpisodeCount = episodeCount
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
	episode := &Episode{}
	err := db.conn.QueryRow(
		`SELECT id, tv_show_id, season_id, season_number, episode_number, title, overview,
			still_path, air_date, runtime, rating, source_id, file_path, file_size, duration,
			video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM episodes WHERE id = ?`,
		id,
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

// GetEpisodeByFilePath retrieves an episode by file path
func (db *DB) GetEpisodeByFilePath(filePath string) (*Episode, error) {
	episode := &Episode{}
	err := db.conn.QueryRow(
		`SELECT id, tv_show_id, season_id, season_number, episode_number, title, overview,
			still_path, air_date, runtime, rating, source_id, file_path, file_size, duration,
			video_codec, audio_codec, resolution, audio_tracks, subtitle_tracks, created_at, updated_at
		 FROM episodes WHERE file_path = ?`,
		filePath,
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
