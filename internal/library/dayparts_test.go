package library

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stephencjuliano/media-server/internal/db"
)

func pool(kind db.MediaType, firstID int64, minutes ...int) []db.LineupItem {
	items := make([]db.LineupItem, len(minutes))
	for i, m := range minutes {
		items[i] = db.LineupItem{ID: firstID + int64(i), Type: kind, Duration: m*60 + i} // +i: no two alike
	}
	return items
}

func TestBuildDaypartCycle(t *testing.T) {
	cartoons := pool(db.MediaTypeEpisode, 100, 22, 22, 23, 22, 21, 22, 24, 22, 22, 23, 22, 22)
	sitcoms := pool(db.MediaTypeEpisode, 200, 24, 22, 44, 22, 23, 22, 25, 22, 45, 22)
	films := pool(db.MediaTypeMovie, 300, 95, 110, 142, 88, 101, 178, 120)
	late := append(pool(db.MediaTypeMovie, 400, 91, 106, 99), pool(db.MediaTypeEpisode, 500, 53, 55, 23, 23, 22)...)

	blocks := []daypartBlock{
		{name: "Morning", from: 6 * 3600, to: 12 * 3600, pool: cartoons},
		{name: "Afternoon", from: 12 * 3600, to: 18 * 3600, pool: sitcoms},
		{name: "Prime", from: 18 * 3600, to: 23 * 3600, pool: films},
		{name: "Late", from: 23 * 3600, to: 6 * 3600, pool: late},
	}
	owner := map[int64]string{}
	for _, block := range blocks {
		for _, item := range block.pool {
			owner[item.ID] = block.name
		}
	}

	const days = 7
	cycle, err := buildDaypartCycle(blocks, days, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}

	// The cycle is exactly a week, so it can loop without drifting.
	total := 0
	for _, item := range cycle {
		total += item.Duration
	}
	if total != days*daySeconds {
		t.Fatalf("cycle runs %ds, want exactly %ds", total, days*daySeconds)
	}

	// Everything starts inside the block it was drawn from, and no block's
	// last programme runs absurdly far past its end.
	cursor := 0
	seenByBlock := map[string]int{}
	for i, item := range cycle {
		timeOfDay := cursor % daySeconds
		var covering daypartBlock
		for _, block := range blocks {
			if block.contains(timeOfDay) {
				covering = block
			}
		}
		if owner[item.ID] != covering.name {
			t.Fatalf("item %d (%s pool) starts at %02d:%02d, in %s", i, owner[item.ID], timeOfDay/3600, timeOfDay/60%60, covering.name)
		}
		seenByBlock[covering.name]++
		cursor += item.Duration
	}
	for _, block := range blocks {
		if seenByBlock[block.name] == 0 {
			t.Errorf("%s never aired", block.name)
		}
	}

	// A film that will not fit the end of prime time waits; an episode-length
	// thing is never passed over for one that overruns by an hour.
	cursor = 0
	for _, item := range cycle {
		timeOfDay := cursor % daySeconds
		if owner[item.ID] == "Afternoon" {
			if past := timeOfDay + item.Duration - 18*3600; past > overrun+45*60 {
				t.Errorf("an afternoon programme runs %d min past six", past/60)
			}
		}
		cursor += item.Duration
	}

	// A block works through its pool before repeating anything.
	firstRepeat := map[int64]int{}
	airedInBlock := map[string]int{}
	for _, item := range cycle {
		name := owner[item.ID]
		if _, again := firstRepeat[item.ID]; !again {
			firstRepeat[item.ID] = airedInBlock[name]
		} else if airedInBlock[name] < len(poolOf(blocks, name))-lookahead {
			t.Errorf("%s repeated item %d after only %d airings of a pool of %d", name, item.ID, airedInBlock[name], len(poolOf(blocks, name)))
			break
		}
		airedInBlock[name]++
	}
}

func poolOf(blocks []daypartBlock, name string) []db.LineupItem {
	for _, block := range blocks {
		if block.name == name {
			return block.pool
		}
	}
	return nil
}

func TestBuildDaypartCycleRejectsABadDay(t *testing.T) {
	some := pool(db.MediaTypeEpisode, 1, 22, 22)
	rng := rand.New(rand.NewSource(1))

	gap := []daypartBlock{{name: "Day", from: 6 * 3600, to: 18 * 3600, pool: some}, {name: "Night", from: 20 * 3600, to: 6 * 3600, pool: some}}
	if _, err := buildDaypartCycle(gap, 1, rng); err == nil {
		t.Error("a day with two uncovered hours was accepted")
	}
	overlap := []daypartBlock{{name: "Day", from: 6 * 3600, to: 20 * 3600, pool: some}, {name: "Night", from: 18 * 3600, to: 6 * 3600, pool: some}}
	if _, err := buildDaypartCycle(overlap, 1, rng); err == nil {
		t.Error("overlapping dayparts were accepted")
	}
	empty := []daypartBlock{{name: "Day", from: 6 * 3600, to: 18 * 3600, pool: some}, {name: "Night", from: 18 * 3600, to: 6 * 3600}}
	if _, err := buildDaypartCycle(empty, 1, rng); err == nil {
		t.Error("a daypart with nothing to air was accepted")
	}
}

func TestParseClock(t *testing.T) {
	for text, want := range map[string]int{"00:00": 0, "06:00": 21600, "23:30": 84600, "24:00": 0, " 6:05 ": 21900} {
		if got, err := parseClock(text); err != nil || got != want {
			t.Errorf("parseClock(%q) = %d, %v; want %d", text, got, err, want)
		}
	}
	for _, text := range []string{"", "6", "25:00", "12:60", "noon", "24:30"} {
		if _, err := parseClock(text); err == nil {
			t.Errorf("parseClock(%q) was accepted", text)
		}
	}
}

func TestApplyLineupWithDayparts(t *testing.T) {
	scanner, database, sources := newMisfiledLibrary(t)
	seedUser(t, database)
	_ = scanner

	for i, title := range []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot"} {
		addFilm(t, database, sources["movies"], title, 1990+i, 85+i*7, 100+i, "Comedy")
	}
	for i, title := range []string{"Scare One", "Scare Two", "Scare Three"} {
		addFilm(t, database, sources["movies"], title, 1980+i, 90+i*5, 200+i, "Horror")
	}

	spec := LineupSpec{
		Timezone: "America/New_York",
		Channels: []LineupChannel{{
			Name: "Movie Night", Days: 2,
			Dayparts: []LineupDaypart{
				{Name: "Day", From: "06:00", To: "22:00", Sources: []LineupSource{{Films: &FilmFilter{Genres: []string{"Comedy"}}}}},
				{Name: "Night", From: "22:00", To: "06:00", Sources: []LineupSource{{Films: &FilmFilter{Genres: []string{"Horror"}}}}},
			},
		}},
	}

	reports, err := ApplyLineup(database, spec, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || len(reports[0].Problems) != 0 || len(reports[0].Dayparts) != 2 || len(reports[0].Films) != 9 {
		t.Fatalf("report = %+v", reports)
	}

	channel, err := database.FindChannelByName("Movie Night")
	if err != nil {
		t.Fatal(err)
	}
	_, seconds, _ := database.ChannelScheduleTotals(channel.ID)
	if seconds != 2*daySeconds {
		t.Errorf("cycle runs %ds, want exactly two days", seconds)
	}

	// Counted from a midnight in the viewer's timezone, or the blocks would
	// sit at the wrong hours.
	newYork, _ := time.LoadLocation("America/New_York")
	anchor := channel.CreatedAt.In(newYork)
	if anchor.Hour() != 0 || anchor.Minute() != 0 || time.Since(anchor) < 0 || time.Since(anchor) > 25*time.Hour {
		t.Errorf("anchored at %v, want today's midnight in New York", anchor)
	}

	// What airs at three in the morning is from the night block.
	var genres string
	err = database.Conn().QueryRow(
		`SELECT m.genres FROM channel_schedule s JOIN media m ON m.id = s.media_id
		 WHERE s.channel_id = ? AND s.cumulative_start <= ? AND s.cumulative_start + s.duration > ?`,
		channel.ID, 27*3600, 27*3600).Scan(&genres)
	if err != nil || genres != "Horror" {
		t.Errorf("at 3am on day two: %q, %v; want a horror film", genres, err)
	}
}
