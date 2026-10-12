package library

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stephencjuliano/media-server/internal/db"
)

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	location, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return location
}

func TestDateWindow(t *testing.T) {
	day := func(month time.Month, d int) time.Time { return time.Date(2026, month, d, 12, 0, 0, 0, time.UTC) }

	december := &DateWindow{From: "12-01", To: "12-25"}
	overNewYear := &DateWindow{From: "12-20", To: "01-02"}
	tests := []struct {
		window *DateWindow
		date   time.Time
		want   bool
	}{
		{december, day(12, 1), true},
		{december, day(12, 25), true},
		{december, day(11, 30), false},
		{december, day(12, 26), false},
		{overNewYear, day(12, 31), true},
		{overNewYear, day(1, 2), true},
		{overNewYear, day(1, 3), false},
		{overNewYear, day(7, 4), false},
		{nil, day(7, 4), true},
		{&DateWindow{From: "Dec 1", To: "12-25"}, day(7, 4), true}, // unreadable: never takes a source off air
	}
	for _, tt := range tests {
		if got := tt.window.includes(tt.date); got != tt.want {
			t.Errorf("%+v includes %s = %v, want %v", tt.window, tt.date.Format("Jan 2"), got, tt.want)
		}
	}
}

func TestClockDriftAcrossAClockChange(t *testing.T) {
	newYork := mustZone(t, "America/New_York")
	// Anchored at a midnight in October, on summer time.
	anchor := time.Date(2026, 10, 11, 0, 0, 0, 0, newYork)

	// Before the clocks go back, the cycle and the clock agree.
	before := time.Date(2026, 10, 31, 21, 30, 0, 0, newYork)
	if drift := clockDrift(anchor, before, newYork); drift != 0 {
		t.Errorf("drift before the change = %s, want none", drift)
	}

	// After, the clock reads an hour less than the cycle has counted.
	after := time.Date(2026, 11, 1, 9, 0, 0, 0, newYork)
	drift := clockDrift(anchor, after, newYork)
	if drift != -time.Hour {
		t.Fatalf("drift after the clocks go back = %s, want -1h", drift)
	}
	// Sliding the anchor by the drift puts the blocks back on their hours.
	if again := clockDrift(anchor.Add(-drift), after, newYork); again != 0 {
		t.Errorf("drift after correcting = %s", again)
	}

	// And forward again in spring.
	spring := time.Date(2027, 3, 14, 9, 0, 0, 0, newYork)
	if drift := clockDrift(anchor.Add(time.Hour), spring, newYork); drift != time.Hour {
		t.Errorf("drift after the clocks go forward = %s, want 1h", drift)
	}
}

func TestKeepLineupCurrent(t *testing.T) {
	_, database, sources := newMisfiledLibrary(t)
	seedUser(t, database)
	newYork := mustZone(t, "America/New_York")

	for i, title := range []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot"} {
		addFilm(t, database, sources["movies"], title, 1990+i, 85+i*7, 100+i, "Comedy")
	}
	for i, title := range []string{"Snow One", "Snow Two"} {
		addFilm(t, database, sources["movies"], title, 1980+i, 90+i*5, 200+i, "Holiday")
	}

	comedy := LineupSource{Films: &FilmFilter{Genres: []string{"Comedy"}}}
	festive := LineupSource{Films: &FilmFilter{Genres: []string{"Holiday"}}, When: &DateWindow{From: "12-01", To: "12-25"}, Repeat: 3}
	spec := LineupSpec{
		Timezone: "America/New_York",
		Channels: []LineupChannel{
			{Name: "Dayparted", Days: 2, Dayparts: []LineupDaypart{
				{Name: "All Day", From: "00:00", To: "24:00", Sources: []LineupSource{comedy, festive}},
			}},
			{Name: "Plain", Sources: []LineupSource{comedy}},
		},
	}
	raw, _ := json.Marshal(spec)

	applied := time.Date(2026, 11, 29, 10, 0, 0, 0, newYork)
	if _, err := ApplyLineupAt(database, spec, true, applied, nil); err != nil {
		t.Fatal(err)
	}
	if err := database.SaveLineupSpec(string(raw)); err != nil {
		t.Fatal(err)
	}

	holidayAirings := func() int {
		var n int
		database.Conn().QueryRow(
			`SELECT count(*) FROM channel_schedule s JOIN channels c ON c.id = s.channel_id JOIN media m ON m.id = s.media_id
			 WHERE c.name = 'Dayparted' AND m.genres = 'Holiday'`).Scan(&n)
		return n
	}
	if holidayAirings() != 0 {
		t.Fatal("holiday films are airing in November")
	}

	// The same day, later: nothing to do, and the day is marked done.
	if done, err := KeepLineupCurrent(database, applied.Add(2*time.Hour)); err != nil || len(done) != 0 {
		t.Errorf("same day: %v, %v; want nothing", done, err)
	}

	// Next morning before the refresh hour: still nothing.
	if done, _ := KeepLineupCurrent(database, time.Date(2026, 11, 30, 3, 0, 0, 0, newYork)); len(done) != 0 {
		t.Errorf("before the refresh hour: %v", done)
	}

	// Two days on, the two-day cycle has run out and gets a new one.
	done, err := KeepLineupCurrent(database, time.Date(2026, 12, 1, 4, 5, 0, 0, newYork))
	if err != nil {
		t.Fatal(err)
	}
	// December 1st is also the day the festive source comes into season.
	if len(done) != 1 || !strings.Contains(done[0], "Dayparted") || !strings.Contains(done[0], "time of year") {
		t.Errorf("on December 1st: %v; want Dayparted rebuilt for the time of year", done)
	}
	if holidayAirings() == 0 {
		t.Error("holiday films are not airing in December")
	}

	// Once a day is enough.
	if again, _ := KeepLineupCurrent(database, time.Date(2026, 12, 1, 9, 0, 0, 0, newYork)); len(again) != 0 {
		t.Errorf("second run on the same day: %v", again)
	}

	// Two days later only the cycle is due.
	done, _ = KeepLineupCurrent(database, time.Date(2026, 12, 3, 4, 5, 0, 0, newYork))
	if len(done) != 1 || !strings.Contains(done[0], "new 2-day cycle") {
		t.Errorf("on December 3rd: %v; want a new cycle", done)
	}

	// Boxing Day: out of season again.
	done, _ = KeepLineupCurrent(database, time.Date(2026, 12, 26, 4, 5, 0, 0, newYork))
	if len(done) != 1 || !strings.Contains(done[0], "time of year") || holidayAirings() != 0 {
		t.Errorf("on December 26th: %v, %d holiday airings", done, holidayAirings())
	}

	// The channel without dayparts or seasons was never rebuilt.
	states, _ := database.ListLineupChannelStates()
	for _, state := range states {
		if state.Name == "Plain" && state.AppliedOn != "2026-11-29" {
			t.Errorf("Plain was re-applied on %s", state.AppliedOn)
		}
	}
}

func TestKeepLineupCurrentFollowsTheClocks(t *testing.T) {
	_, database, sources := newMisfiledLibrary(t)
	seedUser(t, database)
	newYork := mustZone(t, "America/New_York")
	for i, title := range []string{"Alpha", "Bravo", "Charlie", "Delta"} {
		addFilm(t, database, sources["movies"], title, 1990+i, 85+i*7, 100+i, "Comedy")
	}
	spec := LineupSpec{Timezone: "America/New_York", Channels: []LineupChannel{{
		Name: "Dayparted", Days: 30, Dayparts: []LineupDaypart{
			{Name: "All Day", From: "00:00", To: "24:00", Sources: []LineupSource{{Films: &FilmFilter{}}}},
		},
	}}}
	if _, err := ApplyLineupAt(database, spec, true, time.Date(2026, 10, 25, 10, 0, 0, 0, newYork), nil); err != nil {
		t.Fatal(err)
	}

	anchorOf := func() time.Time {
		channel, _ := database.FindChannelByName("Dayparted")
		return channel.CreatedAt
	}
	before := anchorOf()

	// Hallowe'en night: nothing to correct.
	if done, _ := KeepLineupCurrent(database, time.Date(2026, 10, 31, 23, 0, 0, 0, newYork)); len(done) != 0 {
		t.Errorf("before the change: %v", done)
	}
	// The morning after the clocks go back.
	done, err := KeepLineupCurrent(database, time.Date(2026, 11, 1, 3, 0, 0, 0, newYork))
	if err != nil || len(done) != 1 || !strings.Contains(done[0], "follow the clock") {
		t.Fatalf("after the change: %v, %v", done, err)
	}
	if moved := anchorOf().Sub(before); moved != time.Hour {
		t.Errorf("anchor moved %s, want +1h", moved)
	}
	if again, _ := KeepLineupCurrent(database, time.Date(2026, 11, 1, 3, 10, 0, 0, newYork)); len(again) != 0 {
		t.Errorf("corrected twice: %v", again)
	}

	var state db.LineupChannelState
	states, _ := database.ListLineupChannelStates()
	if len(states) == 1 {
		state = states[0]
	}
	if state.Timezone != "America/New_York" || state.CycleDays != 30 {
		t.Errorf("stored state = %+v", state)
	}
}
