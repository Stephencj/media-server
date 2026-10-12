package db

import "fmt"

// GetMoviesInFolder returns the movie rows whose file sits directly in dir.
func (db *DB) GetMoviesInFolder(dir string) ([]*Media, error) {
	prefix := dir + "/"
	rows, err := db.conn.Query(
		`SELECT id, title, original_title, type, year, overview, poster_path, backdrop_path,
			rating, runtime, genres, tmdb_id, imdb_id, season_count, episode_count, source_id,
			file_path, file_size, duration, video_codec, audio_codec, resolution, audio_tracks,
			subtitle_tracks, created_at, updated_at
		 FROM media
		 WHERE type = ? AND substr(file_path, 1, ?) = ? AND instr(substr(file_path, ?), '/') = 0
		 ORDER BY file_path`,
		MediaTypeMovie, len(prefix), prefix, len(prefix)+1,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMediaRows(rows)
}

// RetireMovieRow removes a movie row that turned out to be something else: an
// episode or an extra that now has a row of its own. newID and newType name
// that row; pass newID 0 when there is none.
//
// Whatever pointed at the movie is moved across rather than dropped, so watch
// progress, watchlist and playlist entries and channel schedules survive.
// Where the new row already has an entry of its own, that one is kept.
// Section membership is dropped: sections are filled by rules about movies.
func (db *DB) RetireMovieRow(mediaID, newID int64, newType MediaType) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{"watch_progress", "watchlist", "playlist_items", "channel_schedule"} {
		if newID != 0 {
			if _, err := tx.Exec(
				`UPDATE OR IGNORE `+table+` SET media_id = ?, media_type = ? WHERE media_id = ? AND media_type = ?`,
				newID, newType, mediaID, MediaTypeMovie,
			); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(
			`DELETE FROM `+table+` WHERE media_id = ? AND media_type = ?`, mediaID, MediaTypeMovie,
		); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(
		`DELETE FROM media_sections WHERE media_id = ? AND media_type = ?`, mediaID, MediaTypeMovie,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM media WHERE id = ? AND type = ?`, mediaID, MediaTypeMovie); err != nil {
		return err
	}

	return tx.Commit()
}

// LibraryFile is one playable file, whichever table it lives in.
type LibraryFile struct {
	Kind       MediaType // movie, episode or extra
	ID         int64
	Title      string
	SourceID   int64
	FilePath   string
	FileSize   int64
	Duration   int // seconds
	VideoCodec string
	Resolution string
	// For movies with a TMDB match: the film it is matched to.
	MatchedTitle   string
	MatchedYear    int
	MatchedRuntime int // minutes
	TMDbID         int
	// MatchInDoubt is set while a movie waits in the check-matches queue
	// with another film suggested for it: its match may be the wrong film.
	// A queued movie with no better film on offer is not in doubt in that
	// sense. Its title fits and only its length does not, which points at
	// the file.
	MatchInDoubt bool
}

// ListLibraryFiles returns every movie, episode and extra file.
func (db *DB) ListLibraryFiles() ([]LibraryFile, error) {
	rows, err := db.conn.Query(
		`SELECT 'movie', id, title, COALESCE(source_id, 0), COALESCE(file_path, ''), COALESCE(file_size, 0),
			COALESCE(duration, 0), COALESCE(video_codec, ''), COALESCE(resolution, ''),
			title, COALESCE(year, 0), COALESCE(runtime, 0), COALESCE(tmdb_id, 0),
			EXISTS(SELECT 1 FROM match_reviews r WHERE r.media_id = media.id AND r.status = 'pending'
				AND r.suggested_tmdb_id != 0)
		 FROM media WHERE type = 'movie'
		 UNION ALL
		 SELECT 'episode', id, title, COALESCE(source_id, 0), COALESCE(file_path, ''), COALESCE(file_size, 0),
			COALESCE(duration, 0), COALESCE(video_codec, ''), COALESCE(resolution, ''), '', 0, 0, 0, 0
		 FROM episodes
		 UNION ALL
		 SELECT 'extra', id, title, COALESCE(source_id, 0), COALESCE(file_path, ''), COALESCE(file_size, 0),
			COALESCE(duration, 0), COALESCE(video_codec, ''), COALESCE(resolution, ''), '', 0, 0, 0, 0
		 FROM extras
		 ORDER BY 5`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []LibraryFile
	for rows.Next() {
		var f LibraryFile
		if err := rows.Scan(&f.Kind, &f.ID, &f.Title, &f.SourceID, &f.FilePath, &f.FileSize,
			&f.Duration, &f.VideoCodec, &f.Resolution,
			&f.MatchedTitle, &f.MatchedYear, &f.MatchedRuntime, &f.TMDbID, &f.MatchInDoubt); err != nil {
			return nil, err
		}
		if f.FilePath == "" {
			continue
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// UpdateFileDetails points a movie, episode or extra at a file and records
// that file's technical details. For when a file has been replaced by a
// conversion of itself: the row, and everything that refers to it, stays.
func (db *DB) UpdateFileDetails(kind MediaType, id int64, f *MediaFile) error {
	table := map[MediaType]string{
		MediaTypeMovie:   "media",
		MediaTypeEpisode: "episodes",
		MediaTypeExtra:   "extras",
	}[kind]
	if table == "" {
		return fmt.Errorf("no files are stored for %q", kind)
	}
	_, err := db.conn.Exec(
		`UPDATE `+table+` SET file_path = ?, file_size = ?, duration = ?, video_codec = ?, audio_codec = ?,
			resolution = ?, audio_tracks = ?, subtitle_tracks = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		f.FilePath, f.FileSize, f.Duration, f.VideoCodec, f.AudioCodec,
		f.Resolution, f.AudioTracks, f.SubtitleTracks, id,
	)
	return err
}
