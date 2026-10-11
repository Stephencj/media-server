package db

import (
	"database/sql"
	"time"
)

// Match review statuses.
const (
	MatchReviewPending = "pending" // waiting on a decision
	MatchReviewKept    = "kept"    // looked at, current match confirmed
)

// MatchReview records that a movie's TMDB match looked wrong when it was last
// checked, and what the matcher would use instead.
//
// A kept row is the memory that someone already said no: without it every
// check would raise the same movie again.
type MatchReview struct {
	MediaID         int64
	Status          string
	CurrentTMDbID   int    // the movie's match when it was checked
	SuggestedTMDbID int    // 0 means "no confident match, show the file name"
	SuggestedJSON   string // tmdb.MovieDetails, so accepting needs no network
	Reason          string
	CheckedAt       time.Time
}

// UpsertMatchReview stores the outcome of checking one movie.
func (db *DB) UpsertMatchReview(r *MatchReview) error {
	_, err := db.conn.Exec(
		`INSERT INTO match_reviews (media_id, status, current_tmdb_id, suggested_tmdb_id, suggested_json, reason, checked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(media_id) DO UPDATE SET
		 status = excluded.status, current_tmdb_id = excluded.current_tmdb_id,
		 suggested_tmdb_id = excluded.suggested_tmdb_id, suggested_json = excluded.suggested_json,
		 reason = excluded.reason, checked_at = excluded.checked_at`,
		r.MediaID, r.Status, r.CurrentTMDbID, r.SuggestedTMDbID, r.SuggestedJSON, r.Reason, time.Now(),
	)
	return err
}

// GetMatchReview returns the review for a movie, or ErrNotFound.
func (db *DB) GetMatchReview(mediaID int64) (*MatchReview, error) {
	row := db.conn.QueryRow(
		`SELECT media_id, status, current_tmdb_id, suggested_tmdb_id, suggested_json, reason, checked_at
		 FROM match_reviews WHERE media_id = ?`, mediaID)
	r, err := scanMatchReview(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return r, err
}

// ListPendingMatchReviews returns reviews awaiting a decision, in title order.
// Reviews whose movie has since left the library are skipped.
func (db *DB) ListPendingMatchReviews() ([]*MatchReview, error) {
	rows, err := db.conn.Query(
		`SELECT r.media_id, r.status, r.current_tmdb_id, r.suggested_tmdb_id, r.suggested_json, r.reason, r.checked_at
		 FROM match_reviews r JOIN media m ON m.id = r.media_id
		 WHERE r.status = ? ORDER BY m.title`, MatchReviewPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*MatchReview
	for rows.Next() {
		r, err := scanMatchReview(rows)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}

// SetMatchReviewStatus moves a review to a new status.
func (db *DB) SetMatchReviewStatus(mediaID int64, status string) error {
	_, err := db.conn.Exec(`UPDATE match_reviews SET status = ? WHERE media_id = ?`, status, mediaID)
	return err
}

// DeleteMatchReview forgets a movie's review, once it is resolved or no
// longer applies.
func (db *DB) DeleteMatchReview(mediaID int64) error {
	_, err := db.conn.Exec(`DELETE FROM match_reviews WHERE media_id = ?`, mediaID)
	return err
}

func scanMatchReview(row interface{ Scan(...any) error }) (*MatchReview, error) {
	var r MatchReview
	var suggestedJSON, reason sql.NullString
	if err := row.Scan(&r.MediaID, &r.Status, &r.CurrentTMDbID, &r.SuggestedTMDbID, &suggestedJSON, &reason, &r.CheckedAt); err != nil {
		return nil, err
	}
	r.SuggestedJSON = suggestedJSON.String
	r.Reason = reason.String
	return &r, nil
}
