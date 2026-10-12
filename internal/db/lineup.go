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
// given options: its episodes, and with them its commentaries and bonus clips
// when asked for.
//
// A season limit applies to all of it. A show source on its own limits only
// the episodes, and lets commentaries on every other season through.
func (db *DB) LineupShowItems(showID int64, options *ChannelSourceOptions) ([]LineupItem, error) {
	mode := VersionModeMain
	var seasons []int
	var categories []string
	if options != nil {
		mode = options.GetEffectiveVersionMode()
		seasons = options.Seasons
		categories = append(categories, options.ExtrasCategories...)
	}
	if mode == VersionModeCommentary || mode == VersionModeBoth {
		categories = append(categories, string(ExtraCategoryCommentary))
	}

	inSeasons := func(column string) (string, []any) {
		if len(seasons) == 0 {
			return "", nil
		}
		args := make([]any, len(seasons))
		for i, season := range seasons {
			args[i] = season
		}
		return " AND " + column + " IN (?" + strings.Repeat(", ?", len(seasons)-1) + ")", args
	}

	var items []LineupItem
	collect := func(kind MediaType, query string, args ...any) error {
		rows, err := db.conn.Query(query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item := LineupItem{Type: kind}
			if err := rows.Scan(&item.ID, &item.Title, &item.Duration); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	}

	if mode == VersionModeMain || mode == VersionModeBoth {
		clause, args := inSeasons("season_number")
		if err := collect(MediaTypeEpisode,
			`SELECT id, title, duration FROM episodes WHERE tv_show_id = ? AND duration > 0`+clause+
				` ORDER BY season_number, episode_number`,
			append([]any{showID}, args...)...); err != nil {
			return nil, err
		}
	}
	if len(categories) > 0 {
		// Bonus material filed under no season cannot be placed, so a
		// season limit leaves it out rather than guess.
		clause, args := inSeasons("season_number")
		query := `SELECT id, title, duration FROM extras WHERE tv_show_id = ? AND duration > 0
			AND category IN (?` + strings.Repeat(", ?", len(categories)-1) + `)` + clause + ` ORDER BY season_number, episode_number, title`
		params := []any{showID}
		for _, category := range categories {
			params = append(params, category)
		}
		if err := collect(MediaTypeExtra, query, append(params, args...)...); err != nil {
			return nil, err
		}
	}
	return items, nil
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

// LineupPlaylistItems returns what a playlist holds, by the playlist's name,
// whoever owns it. found is false when there is no playlist of that name.
func (db *DB) LineupPlaylistItems(name string) (items []LineupItem, found bool, err error) {
	var playlistID int64
	err = db.conn.QueryRow(`SELECT id FROM playlists WHERE name = ? ORDER BY id LIMIT 1`, name).Scan(&playlistID)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	for _, input := range db.getMediaFromSource(ChannelSource{SourceType: ChannelSourcePlaylist, SourceID: &playlistID}) {
		items = append(items, LineupItem{ID: input.MediaID, Type: input.MediaType, Title: input.Title, Duration: input.Duration})
	}
	return items, true, nil
}

// SaveLineupSpec remembers the lineup as applied, so it can be re-applied
// without the file.
func (db *DB) SaveLineupSpec(spec string) error {
	_, err := db.conn.Exec(
		`INSERT INTO lineup_state (id, spec) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET spec = excluded.spec`, spec)
	return err
}

// LoadLineupSpec returns the remembered lineup and the date it was last
// refreshed ("2006-01-02", empty if never). spec is empty when none has been
// applied.
func (db *DB) LoadLineupSpec() (spec, refreshedOn string, err error) {
	err = db.conn.QueryRow(`SELECT spec, refreshed_on FROM lineup_state WHERE id = 1`).Scan(&spec, &refreshedOn)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return spec, refreshedOn, err
}

// MarkLineupRefreshed records the date of the daily refresh.
func (db *DB) MarkLineupRefreshed(date string) error {
	_, err := db.conn.Exec(`UPDATE lineup_state SET refreshed_on = ? WHERE id = 1`, date)
	return err
}

// LineupChannelState is what the lineup last did for a channel.
type LineupChannelState struct {
	ChannelID int64
	Name      string
	Anchor    time.Time // the channel's created_at, which its cycle counts from
	// Timezone is set for a channel with dayparts: the clock its blocks
	// follow.
	Timezone string
	// Signature says which of the channel's seasonal sources were in
	// season when it was applied.
	Signature string
	CycleDays int
	AppliedOn string // "2006-01-02"
}

// SetLineupChannelState records how a channel was just applied.
func (db *DB) SetLineupChannelState(state LineupChannelState) error {
	_, err := db.conn.Exec(
		`INSERT INTO lineup_channels (channel_id, timezone, signature, cycle_days, applied_on) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(channel_id) DO UPDATE SET timezone = excluded.timezone, signature = excluded.signature,
		 cycle_days = excluded.cycle_days, applied_on = excluded.applied_on`,
		state.ChannelID, state.Timezone, state.Signature, state.CycleDays, state.AppliedOn,
	)
	return err
}

// ListLineupChannelStates returns the state of every channel the lineup
// manages.
func (db *DB) ListLineupChannelStates() ([]LineupChannelState, error) {
	rows, err := db.conn.Query(
		`SELECT l.channel_id, c.name, c.created_at, l.timezone, l.signature, l.cycle_days, l.applied_on
		 FROM lineup_channels l JOIN channels c ON c.id = l.channel_id ORDER BY c.channel_number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []LineupChannelState
	for rows.Next() {
		var s LineupChannelState
		if err := rows.Scan(&s.ChannelID, &s.Name, &s.Anchor, &s.Timezone, &s.Signature, &s.CycleDays, &s.AppliedOn); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, rows.Err()
}

// SetChannelAnchor moves the moment a channel's cycle counts from, which
// slides what is airing by the same amount.
func (db *DB) SetChannelAnchor(channelID int64, anchor time.Time) error {
	_, err := db.conn.Exec(`UPDATE channels SET created_at = ? WHERE id = ?`, anchor.UTC(), channelID)
	return err
}
