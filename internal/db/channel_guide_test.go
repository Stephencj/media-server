package db

import (
	"path/filepath"
	"testing"
	"time"
)

// newTestDB spins up a migrated database in a temp dir.
func newTestDB(t *testing.T) *DB {
	t.Helper()
	database, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func newTestUser(t *testing.T, database *DB) int64 {
	t.Helper()
	res, err := database.conn.Exec(
		`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
		"tester", "tester@example.com", "x",
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// seedCycle writes a cycle of items with the given durations (seconds) and
// pins the channel's epoch to createdAt, which is what the guide projects from.
func seedCycle(t *testing.T, database *DB, channelID int64, createdAt time.Time, durations []int) {
	t.Helper()
	cumulative := 0
	for i, d := range durations {
		_, err := database.conn.Exec(
			`INSERT INTO channel_schedule
				(channel_id, media_id, media_type, scheduled_position, cycle_number, duration, cumulative_start, played)
			VALUES (?, ?, 'movie', ?, 1, ?, ?, 0)`,
			channelID, int64(1000+i), i, d, cumulative,
		)
		if err != nil {
			t.Fatalf("insert schedule: %v", err)
		}
		cumulative += d
	}
	if _, err := database.conn.Exec(
		`UPDATE channels SET created_at = ? WHERE id = ?`, createdAt, channelID,
	); err != nil {
		t.Fatalf("set created_at: %v", err)
	}
}

func TestChannelNumbersAssignedSequentially(t *testing.T) {
	database := newTestDB(t)
	userID := newTestUser(t, database)

	first, err := database.CreateChannel(userID, "Movies", "", "🎬")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	second, err := database.CreateChannel(userID, "Sitcoms", "", "📺")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if first.ChannelNumber != 101 {
		t.Errorf("first channel number = %d, want 101", first.ChannelNumber)
	}
	if second.ChannelNumber != 102 {
		t.Errorf("second channel number = %d, want 102", second.ChannelNumber)
	}

	// Listing must come back in guide order.
	channels, err := database.GetUserChannels(userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(channels) != 2 || channels[0].ChannelNumber != 101 || channels[1].ChannelNumber != 102 {
		t.Errorf("channels out of guide order: %+v", channels)
	}
}

func TestGuideProjectsContiguousPrograms(t *testing.T) {
	database := newTestDB(t)
	userID := newTestUser(t, database)

	channel, err := database.CreateChannel(userID, "Movies", "", "🎬")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// 90-minute cycle: 30 + 40 + 20.
	epoch := time.Now().Add(-10 * time.Hour).UTC()
	seedCycle(t, database, channel.ID, epoch, []int{1800, 2400, 1200})

	start := time.Now().UTC().Truncate(30 * time.Minute)
	end := start.Add(4 * time.Hour)

	guide, err := database.GetChannelGuide(userID, start, end)
	if err != nil {
		t.Fatalf("guide: %v", err)
	}
	if len(guide.Channels) != 1 {
		t.Fatalf("got %d channels, want 1", len(guide.Channels))
	}

	programs := guide.Channels[0].Programs
	if len(programs) == 0 {
		t.Fatal("no programs projected")
	}

	// A 4-hour window over a 90-minute cycle must wrap: more airings than the
	// cycle has items. This is the case the old up_next query could not serve.
	if len(programs) <= 3 {
		t.Errorf("got %d programs over 4h of a 90m cycle, expected the cycle to repeat", len(programs))
	}

	// The first program must already be airing when the window opens.
	if programs[0].StartTime.After(start) {
		t.Errorf("first program starts at %v, after window start %v", programs[0].StartTime, start)
	}

	// No gaps, no overlaps, and every duration honoured.
	for i, p := range programs {
		wantEnd := p.StartTime.Add(time.Duration(p.Duration) * time.Second)
		if !p.EndTime.Equal(wantEnd) {
			t.Errorf("program %d: end %v != start+duration %v", i, p.EndTime, wantEnd)
		}
		if i > 0 && !p.StartTime.Equal(programs[i-1].EndTime) {
			t.Errorf("program %d starts %v but previous ended %v", i, p.StartTime, programs[i-1].EndTime)
		}
	}

	// The window must be fully covered.
	if last := programs[len(programs)-1]; last.EndTime.Before(end) {
		t.Errorf("coverage ends at %v, before window end %v", last.EndTime, end)
	}
}

// The projected "now" must agree with what GetChannelNowPlaying reports,
// otherwise the guide and the player would disagree about what is on.
func TestGuideAgreesWithNowPlaying(t *testing.T) {
	database := newTestDB(t)
	userID := newTestUser(t, database)

	channel, err := database.CreateChannel(userID, "Movies", "", "🎬")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	epoch := time.Now().Add(-7*time.Hour - 13*time.Minute).UTC()
	seedCycle(t, database, channel.ID, epoch, []int{1800, 2400, 1200})

	now := time.Now().UTC()
	guide, err := database.GetChannelGuide(userID, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("guide: %v", err)
	}

	nowPlaying, err := database.GetChannelNowPlaying(channel.ID)
	if err != nil {
		t.Fatalf("now playing: %v", err)
	}
	if nowPlaying.NowPlaying == nil {
		t.Fatal("now playing is nil")
	}

	var airing *GuideProgram
	for i := range guide.Channels[0].Programs {
		p := &guide.Channels[0].Programs[i]
		if !p.StartTime.After(now) && p.EndTime.After(now) {
			airing = p
			break
		}
	}
	if airing == nil {
		t.Fatal("guide has no program airing now")
	}

	if airing.MediaID != nowPlaying.NowPlaying.MediaID {
		t.Errorf("guide says media %d is on, now-playing says %d",
			airing.MediaID, nowPlaying.NowPlaying.MediaID)
	}
}

// A channel near the end of its cycle must still report a full up_next list.
func TestUpNextWrapsAtEndOfCycle(t *testing.T) {
	database := newTestDB(t)
	userID := newTestUser(t, database)

	channel, err := database.CreateChannel(userID, "Movies", "", "🎬")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Four items, 1800s each (2h cycle). Put the channel inside the LAST item
	// by making it 7200-900 = 6300s old: position 6300 falls in item index 3.
	epoch := time.Now().Add(-6300 * time.Second).UTC()
	seedCycle(t, database, channel.ID, epoch, []int{1800, 1800, 1800, 1800})

	nowPlaying, err := database.GetChannelNowPlaying(channel.ID)
	if err != nil {
		t.Fatalf("now playing: %v", err)
	}
	if nowPlaying.NowPlaying == nil {
		t.Fatal("now playing is nil")
	}
	if nowPlaying.NowPlaying.ScheduledPosition != 3 {
		t.Fatalf("expected to be in the last item, got position %d",
			nowPlaying.NowPlaying.ScheduledPosition)
	}

	// Before the wrap fix this returned zero items and playback dead-ended.
	if len(nowPlaying.UpNext) != 3 {
		t.Fatalf("up_next has %d items at end of cycle, want 3", len(nowPlaying.UpNext))
	}
	for i, want := range []int{0, 1, 2} {
		if nowPlaying.UpNext[i].ScheduledPosition != want {
			t.Errorf("up_next[%d] position = %d, want %d (wrapped to cycle start)",
				i, nowPlaying.UpNext[i].ScheduledPosition, want)
		}
	}
}

// An empty channel keeps its lane so the guide's rows stay stable.
func TestGuideKeepsEmptyChannels(t *testing.T) {
	database := newTestDB(t)
	userID := newTestUser(t, database)

	if _, err := database.CreateChannel(userID, "Empty", "", "📺"); err != nil {
		t.Fatalf("create: %v", err)
	}

	now := time.Now().UTC()
	guide, err := database.GetChannelGuide(userID, now, now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("guide: %v", err)
	}
	if len(guide.Channels) != 1 {
		t.Fatalf("got %d channels, want the empty one kept", len(guide.Channels))
	}
	if len(guide.Channels[0].Programs) != 0 {
		t.Errorf("empty channel should have no programs, got %d", len(guide.Channels[0].Programs))
	}
}
