package library

import (
	"testing"

	"github.com/stephencjuliano/media-server/internal/db"
)

func addFilm(t *testing.T, database *db.DB, source *db.MediaSource, title string, year, minutes, tmdbID int, genres string) *db.Media {
	t.Helper()
	media := &db.Media{Type: db.MediaTypeMovie}
	media.SourceID = source.ID
	media.FilePath = "/media/Movies/" + title + "/" + title + ".m4v"
	media.Duration = minutes * 60
	media.VideoCodec = "h264"
	media.Title, media.Year, media.TMDbID, media.Genres = title, year, tmdbID, genres
	created, err := database.CreateMedia(media)
	if err != nil {
		t.Fatalf("create film: %v", err)
	}
	return created
}

func TestApplyLineup(t *testing.T) {
	scanner, database, sources := newMisfiledLibrary(t)
	seedUser(t, database)

	scream := addFilm(t, database, sources["movies"], "Scream", 1996, 110, 1, "Crime, Horror, Mystery")
	addFilm(t, database, sources["movies"], "Gremlins", 1984, 106, 2, "Fantasy, Horror, Comedy")
	addFilm(t, database, sources["movies"], "Charade", 1963, 116, 3, "Comedy, Mystery, Romance")
	addFilm(t, database, sources["movies"], "Frosty the Snowman", 1969, 25, 4, "Family, Animation, TV Movie")
	addFilm(t, database, sources["movies"], "Unmatched Thing", 0, 90, 0, "")

	// A show with two episodes, and a commentary on a film, by way of the
	// scanner so the rows are the shape the server makes.
	addMovieRow(t, database, sources["tv"], episodePath, "x", 45)
	if err := scanner.processFile(episodePath, sources["tv"]); err != nil {
		t.Fatal(err)
	}
	addMovieRow(t, database, sources["movies"], "/media/Movies/Scream/Scream Commentary.m4v", "x", 110)
	if err := scanner.processFile("/media/Movies/Scream/Scream Commentary.m4v", sources["movies"]); err != nil {
		t.Fatal(err)
	}

	spec := LineupSpec{Channels: []LineupChannel{
		{Name: "Fright Night", Sources: []LineupSource{{Films: &FilmFilter{Genres: []string{"horror"}}}}},
		{Name: "Golden Age", Sources: []LineupSource{{Films: &FilmFilter{YearTo: 1969, MinMinutes: 60}}}},
		{Name: "Hand Picked", Number: 100, Sources: []LineupSource{{Films: &FilmFilter{Titles: []string{"charade"}}}}},
		{Name: "Movie Night", Sources: []LineupSource{{Films: &FilmFilter{MinMinutes: 60}}}},
		{Name: "Mixed", Sources: []LineupSource{
			{Show: "Taskmaster"},
			{Bonus: &BonusFilter{Categories: []string{"commentary"}, Parent: "film"}},
			{Show: "No Such Show"},
		}},
		{Name: "Empty", Sources: []LineupSource{{Films: &FilmFilter{Genres: []string{"Western"}}}}},
	}}

	plan, err := ApplyLineup(database, spec, false)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]LineupReport{}
	for _, report := range plan {
		byName[report.Channel] = report
	}
	for name, want := range map[string]int{"Fright Night": 2, "Golden Age": 1, "Hand Picked": 1, "Movie Night": 3} {
		if got := len(byName[name].Films); got != want {
			t.Errorf("%s: %d films %v, want %d", name, got, byName[name].Films, want)
		}
	}
	if mixed := byName["Mixed"]; len(mixed.Shows) != 1 || mixed.Bonus != 1 || len(mixed.Problems) != 1 {
		t.Errorf("Mixed = %+v", mixed)
	}
	if _, err := database.FindChannelByName("Fright Night"); err != db.ErrNotFound {
		t.Fatal("planning created a channel")
	}

	// Apply twice: the second pass must refresh, not duplicate.
	for pass := 1; pass <= 2; pass++ {
		done, err := ApplyLineup(database, spec, true)
		if err != nil {
			t.Fatalf("pass %d: %v", pass, err)
		}
		for _, report := range done {
			wantAction := map[int]string{1: "create", 2: "update"}[pass]
			if report.Channel == "Empty" {
				if report.Items != 0 {
					t.Errorf("Empty was scheduled")
				}
				continue
			}
			if report.Action != wantAction || report.Items == 0 || report.Hours == 0 {
				t.Errorf("pass %d: %+v", pass, report)
			}
		}
	}

	var channels, playlists int
	database.Conn().QueryRow(`SELECT count(*) FROM channels`).Scan(&channels)
	database.Conn().QueryRow(`SELECT count(*) FROM playlists`).Scan(&playlists)
	if channels != 5 || playlists != 5 {
		t.Errorf("%d channels and %d playlists, want 5 and 5", channels, playlists)
	}
	if _, err := database.FindChannelByName("Empty"); err != db.ErrNotFound {
		t.Error("a channel with nothing to air was created")
	}

	if picked, _ := database.FindChannelByName("Hand Picked"); picked.ChannelNumber != 100 {
		t.Errorf("Hand Picked is channel %d, want the 100 the file gives it", picked.ChannelNumber)
	}

	fright, _ := database.FindChannelByName("Fright Night")
	var inSchedule int
	database.Conn().QueryRow(
		`SELECT count(*) FROM channel_schedule WHERE channel_id = ? AND media_type = 'movie' AND media_id = ?`,
		fright.ID, scream.ID).Scan(&inSchedule)
	if inSchedule == 0 {
		t.Error("Scream is not on Fright Night")
	}

	mixed, _ := database.FindChannelByName("Mixed")
	var kinds int
	database.Conn().QueryRow(`SELECT count(DISTINCT media_type) FROM channel_schedule WHERE channel_id = ?`, mixed.ID).Scan(&kinds)
	if kinds != 2 {
		t.Errorf("Mixed airs %d kinds of thing, want episodes and a commentary", kinds)
	}
}
