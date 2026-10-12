package db

import (
	"database/sql"
	"strings"
	"time"
)

// LineupItem is one thing a channel can air.
type LineupItem struct {
	ID       int64
	Type     MediaType
	Title    string
	Duration int // seconds
}

// ChannelOwner returns the user new channels should belong to: whoever owns
// the most channels already, or the first user when there are none.
func (db *DB) ChannelOwner() (int64, error) {
	var userID int64
	err := db.conn.QueryRow(
		`SELECT user_id FROM channels GROUP BY user_id ORDER BY count(*) DESC, user_id LIMIT 1`,
	).Scan(&userID)
	if err == nil {
		return userID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	user, err := db.FirstUser()
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

// FindChannelByName returns the channel with this name, or ErrNotFound.
func (db *DB) FindChannelByName(name string) (*Channel, error) {
	var id int64
	err := db.conn.QueryRow(`SELECT id FROM channels WHERE name = ? ORDER BY id LIMIT 1`, name).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return db.GetChannelByID(id)
}

// ClearChannelSources removes every source from a channel, ahead of giving
// it a new set.
func (db *DB) ClearChannelSources(channelID int64) error {
	_, err := db.conn.Exec(`DELETE FROM channel_sources WHERE channel_id = ?`, channelID)
	return err
}

// EnsurePlaylist returns the ID of the user's playlist with this name,
// creating it if there is none.
func (db *DB) EnsurePlaylist(userID int64, name, description string) (int64, error) {
	var id int64
	err := db.conn.QueryRow(`SELECT id FROM playlists WHERE user_id = ? AND name = ?`, userID, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	result, err := db.conn.Exec(
		`INSERT INTO playlists (user_id, name, description) VALUES (?, ?, ?)`, userID, name, description,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// ReplacePlaylistItems makes a playlist hold exactly these items, in order.
func (db *DB) ReplacePlaylistItems(playlistID int64, items []LineupItem) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM playlist_items WHERE playlist_id = ?`, playlistID); err != nil {
		return err
	}
	for position, item := range items {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO playlist_items (playlist_id, media_id, media_type, position) VALUES (?, ?, ?, ?)`,
			playlistID, item.ID, item.Type, position,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// LineupExtras returns extras of the given categories. parent narrows them to
// those belonging to a "film" or a "show"; anything else means either.
func (db *DB) LineupExtras(categories []string, parent string) ([]LineupItem, error) {
	if len(categories) == 0 {
		return nil, nil
	}
	query := `SELECT id, title, COALESCE(duration, 0) FROM extras
		WHERE duration > 0 AND category IN (?` + strings.Repeat(", ?", len(categories)-1) + `)`
	switch parent {
	case "film":
		query += ` AND movie_id IS NOT NULL`
	case "show":
		query += ` AND (tv_show_id IS NOT NULL OR episode_id IS NOT NULL)`
	}
	query += ` ORDER BY title`

	args := make([]any, len(categories))
	for i, category := range categories {
		args[i] = category
	}
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []LineupItem
	for rows.Next() {
		item := LineupItem{Type: MediaTypeExtra}
		if err := rows.Scan(&item.ID, &item.Title, &item.Duration); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ChannelScheduleTotals returns how many items a channel's cycle holds and
// how long it runs, in seconds.
func (db *DB) ChannelScheduleTotals(channelID int64) (items, seconds int, err error) {
	err = db.conn.QueryRow(
		`SELECT count(*), COALESCE(sum(duration), 0) FROM channel_schedule WHERE channel_id = ?`, channelID,
	).Scan(&items, &seconds)
	return items, seconds, err
}

// LineupShowItems returns what a show contributes to a channel under the
// given options (seasons, commentaries, bonus clips): the same items a show
// source would schedule.
func (db *DB) LineupShowItems(showID int64, options *ChannelSourceOptions) []LineupItem {
	inputs := db.getMediaFromSource(ChannelSource{SourceType: ChannelSourceShow, SourceID: &showID, Options: options})
	items := make([]LineupItem, 0, len(inputs))
	for _, input := range inputs {
		items = append(items, LineupItem{ID: input.MediaID, Type: input.MediaType, Title: input.Title, Duration: input.Duration})
	}
	return items
}

// ReplaceChannelSchedule gives a channel exactly this cycle, in order, and
// anchors it: the cycle's first second airs at anchor, and every later moment
// follows from there. A cycle that is a whole number of days long and
// anchored at a midnight therefore keeps the same time of day at the same
// point in every pass.
func (db *DB) ReplaceChannelSchedule(channelID int64, items []LineupItem, anchor time.Time) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM channel_schedule WHERE channel_id = ?`, channelID); err != nil {
		return err
	}
	start := 0
	for position, item := range items {
		if _, err := tx.Exec(
			`INSERT INTO channel_schedule (channel_id, media_id, media_type, scheduled_position, cycle_number, duration, cumulative_start)
			 VALUES (?, ?, ?, ?, 1, ?, ?)`,
			channelID, item.ID, item.Type, position, item.Duration, start,
		); err != nil {
			return err
		}
		start += item.Duration
	}
	// A channel's place in its cycle is counted from when it was created.
	if _, err := tx.Exec(`UPDATE channels SET created_at = ? WHERE id = ?`, anchor.UTC(), channelID); err != nil {
		return err
	}
	return tx.Commit()
}

// SetChannelNumber gives a channel its number in the guide, which is also
// its place in the order.
func (db *DB) SetChannelNumber(channelID int64, number int) error {
	_, err := db.conn.Exec(`UPDATE channels SET channel_number = ? WHERE id = ?`, number, channelID)
	return err
}
