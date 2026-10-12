package library

import (
	"path/filepath"
	"testing"

	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
)

// A library the way an earlier scan left it: an episode and a commentary
// track both filed as movies, beside the one real film.
func newMisfiledLibrary(t *testing.T) (*Scanner, *db.DB, map[string]*db.MediaSource) {
	t.Helper()
	database, err := db.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	sources := map[string]*db.MediaSource{}
	for name, path := range map[string]string{"tv": "/media/TV Shows", "movies": "/media/Movies"} {
		source, err := database.CreateMediaSource(&db.MediaSource{Name: name, Path: path, Type: "local", Enabled: true})
		if err != nil {
			t.Fatalf("create source: %v", err)
		}
		sources[name] = source
	}

	return NewScanner(database, &config.Config{}), database, sources
}

func addMovieRow(t *testing.T, database *db.DB, source *db.MediaSource, path, title string, minutes int) *db.Media {
	t.Helper()
	media := &db.Media{Type: db.MediaTypeMovie}
	media.SourceID = source.ID
	media.FilePath = path
	media.Duration = minutes * 60
	media.VideoCodec = "h264"
	media.Title = title
	created, err := database.CreateMedia(media)
	if err != nil {
		t.Fatalf("create media: %v", err)
	}
	return created
}

const (
	episodePath    = "/media/TV Shows/Taskmaster/Series 04/01 A Fat Bald White Man (1080p HD).m4v"
	filmPath       = "/media/Movies/Gremlins/Gremlins.m4v"
	commentaryPath = "/media/Movies/Gremlins/Gremlins Commentary 2.m4v"
)

func TestScanMovesMisfiledMoviesWhereTheirFoldersSay(t *testing.T) {
	scanner, database, sources := newMisfiledLibrary(t)

	episodeRow := addMovieRow(t, database, sources["tv"], episodePath, "A Fat Bald White Man ( )", 45)
	film := addMovieRow(t, database, sources["movies"], filmPath, "Gremlins", 106)
	addMovieRow(t, database, sources["movies"], commentaryPath, "Gremlins Commentary 2", 106)

	// Someone was partway through the episode while it was filed as a movie.
	userID := seedUser(t, database)
	if _, err := database.Conn().Exec(
		`INSERT INTO watch_progress (user_id, media_id, media_type, position, duration) VALUES (?, ?, 'movie', 600, 2700)`,
		userID, episodeRow.ID); err != nil {
		t.Fatalf("seed progress: %v", err)
	}

	steps, err := scanner.PlanTidy()
	if err != nil || len(steps) != 2 {
		t.Fatalf("PlanTidy = %+v, %v; want 2 steps", steps, err)
	}
	if movies, _ := database.GetMediaByType(db.MediaTypeMovie, 100, 0); len(movies) != 3 {
		t.Fatalf("PlanTidy changed the library: %d movies", len(movies))
	}

	// Twice: the second pass must find nothing left to do.
	for pass := 1; pass <= 2; pass++ {
		// The commentary first, as a directory walk would deliver it.
		for _, c := range []struct{ source, path string }{
			{"movies", commentaryPath}, {"movies", filmPath}, {"tv", episodePath},
		} {
			if err := scanner.processFile(c.path, sources[c.source]); err != nil {
				t.Fatalf("pass %d: processFile(%s): %v", pass, c.path, err)
			}
		}
	}

	movies, _ := database.GetMediaByType(db.MediaTypeMovie, 100, 0)
	if len(movies) != 1 || movies[0].ID != film.ID {
		t.Fatalf("movies after scan = %d, want only the film", len(movies))
	}

	episode, err := database.GetEpisodeByFilePath(episodePath)
	if err != nil {
		t.Fatalf("episode not created: %v", err)
	}
	if episode.SeasonNumber != 4 || episode.EpisodeNumber != 1 || episode.Title != "A Fat Bald White Man" || episode.Duration != 45*60 {
		t.Errorf("episode = S%02dE%02d %q, %ds", episode.SeasonNumber, episode.EpisodeNumber, episode.Title, episode.Duration)
	}
	if show, err := database.GetTVShowByTitle("Taskmaster"); err != nil || show.ID != episode.TVShowID {
		t.Errorf("episode not under Taskmaster: %v", err)
	}

	extra, err := database.GetExtraByFilePath(commentaryPath)
	if err != nil {
		t.Fatalf("extra not created: %v", err)
	}
	if extra.Category != db.ExtraCategoryCommentary || extra.MovieID == nil || *extra.MovieID != film.ID {
		t.Errorf("extra = %s, movie %v; want a commentary on the film", extra.Category, extra.MovieID)
	}

	var mediaID int64
	var mediaType string
	if err := database.Conn().QueryRow(`SELECT media_id, media_type FROM watch_progress WHERE user_id = ?`, userID).
		Scan(&mediaID, &mediaType); err != nil {
		t.Fatalf("watch progress was lost: %v", err)
	}
	if mediaID != episode.ID || mediaType != "episode" {
		t.Errorf("watch progress points at %s %d, want the episode", mediaType, mediaID)
	}

	if steps, _ := scanner.PlanTidy(); len(steps) != 0 {
		t.Errorf("still %d steps planned after the scan", len(steps))
	}
}

func TestScanRemovesMovieRowsThatDuplicateAnEpisode(t *testing.T) {
	scanner, database, sources := newMisfiledLibrary(t)

	// The file is already an episode, and also a stale movie row.
	if err := scanner.processFile(episodePath, sources["tv"]); err == nil {
		t.Fatal("expected the first scan to fail: there is no file to probe")
	}
	addMovieRow(t, database, sources["tv"], episodePath, "A Fat Bald White Man ( )", 45)
	if err := scanner.processFile(episodePath, sources["tv"]); err != nil {
		t.Fatalf("processFile: %v", err)
	}
	episode, err := database.GetEpisodeByFilePath(episodePath)
	if err != nil {
		t.Fatalf("episode missing: %v", err)
	}

	addMovieRow(t, database, sources["tv"], episodePath, "A Fat Bald White Man ( )", 45)
	steps, _ := scanner.PlanTidy()
	if len(steps) != 1 || steps[0].Action != "remove duplicate" {
		t.Fatalf("plan = %+v, want one duplicate to remove", steps)
	}
	if err := scanner.processFile(episodePath, sources["tv"]); err != nil {
		t.Fatalf("processFile: %v", err)
	}
	if movies, _ := database.GetMediaByType(db.MediaTypeMovie, 100, 0); len(movies) != 0 {
		t.Errorf("%d movie rows left", len(movies))
	}
	if again, err := database.GetEpisodeByFilePath(episodePath); err != nil || again.ID != episode.ID {
		t.Errorf("the episode itself changed: %v", err)
	}
}

func seedUser(t *testing.T, database *db.DB) int64 {
	t.Helper()
	res, err := database.Conn().Exec(
		`INSERT INTO users (username, email, password_hash) VALUES ('tester', 'tester@example.com', 'x')`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestCheckFileHealth(t *testing.T) {
	sources := map[int64]string{1: "/media/Movies", 2: "/media/TV Shows"}
	const mb = 1 << 20

	movie := func(path, matched string, year, minutes, runtime int) db.LibraryFile {
		return db.LibraryFile{
			Kind: db.MediaTypeMovie, SourceID: 1, FilePath: path, Duration: minutes * 60, VideoCodec: "h264",
			FileSize: int64(minutes) * 30 * mb, MatchedTitle: matched, MatchedYear: year, MatchedRuntime: runtime, TMDbID: 1,
		}
	}

	files := []db.LibraryFile{
		movie("/media/Movies/Gremlins/Gremlins.m4v", "Gremlins", 1984, 106, 106),
		// Half the film is missing.
		movie("/media/Movies/Scott Pilgrim vs the World/Scott Pilgrim.m4v", "Scott Pilgrim vs. the World", 2010, 51, 112),
		// A little short.
		movie("/media/Movies/Charade (1963).mkv", "Charade", 1963, 98, 113),
		// Short, but the match itself is wrong, so the length proves nothing.
		movie("/media/Movies/Road Hard/Road Hard.mp4", "Hard Road", 1988, 60, 120),
		// Three copies of one film.
		movie("/media/Movies/Shrek 2.m4v", "Shrek 2", 2004, 93, 93),
		movie("/media/Movies/Shrek 2 HD.m4v", "Shrek 2", 2004, 93, 93),
		movie("/media/Movies/Shrek 2 T00.mp4", "Shrek 2", 2004, 93, 93),
		// A name the converter cut off.
		movie("/media/Movies/04 How to Train Your Dragon (1080p H.m4v", "How to Train Your Dragon", 2010, 98, 98),
		// A commentary is as long as its film and is no copy of it.
		movie("/media/Movies/Gremlins/Gremlins Commentary 2.m4v", "", 0, 106, 0),
		{Kind: db.MediaTypeEpisode, SourceID: 2, FilePath: "/media/TV Shows/Some Show/S01E01 - Pilot.m4v", VideoCodec: "h264"},
		{Kind: db.MediaTypeEpisode, SourceID: 2, FilePath: "/media/TV Shows/Some Show/S01E02 - Two.m4v", Duration: 1300, FileSize: 400 * mb},
		{Kind: db.MediaTypeExtra, SourceID: 2, FilePath: "/media/TV Shows/Some Show/S01D01 Play All.mkv", Duration: 9000, VideoCodec: "mpeg2", FileSize: 4000 * mb},
		{Kind: db.MediaTypeExtra, SourceID: 2, FilePath: "/media/TV Shows/Some Show/Thing_preview.m4v", Duration: 60, VideoCodec: "h264", FileSize: 5 * mb},
		{Kind: db.MediaTypeEpisode, SourceID: 2, FilePath: "/media/TV Shows/Some Show/S01E03 - Starved.m4v", Duration: 1300, VideoCodec: "h264", FileSize: 20 * mb},
	}

	got := map[string]FileIssue{}
	for _, issue := range CheckFileHealth(files, sources) {
		got[filepath.Base(issue.Path)] = issue
	}

	want := map[string]string{
		"Scott Pilgrim.m4v":                        SeverityDefinite,
		"S01E01 - Pilot.m4v":                       SeverityDefinite, // no running time
		"S01E02 - Two.m4v":                         SeverityDefinite, // no video stream
		"Charade (1963).mkv":                       SeverityPotential,
		"Shrek 2.m4v":                              SeverityPotential,
		"Shrek 2 HD.m4v":                           SeverityPotential,
		"Shrek 2 T00.mp4":                          SeverityPotential,
		"04 How to Train Your Dragon (1080p H.m4v": SeverityPotential,
		"S01D01 Play All.mkv":                      SeverityPotential,
		"Thing_preview.m4v":                        SeverityPotential,
		"S01E03 - Starved.m4v":                     SeverityPotential,
	}
	for name, severity := range want {
		if got[name].Severity != severity {
			t.Errorf("%s: severity %q (%v), want %q", name, got[name].Severity, got[name].Reasons, severity)
		}
	}
	for name, issue := range got {
		if _, expected := want[name]; !expected {
			t.Errorf("%s: flagged %s (%v), want it left alone", name, issue.Severity, issue.Reasons)
		}
	}

	if issues := CheckFileHealth(files, sources); len(issues) > 0 && issues[0].Severity != SeverityDefinite {
		t.Errorf("definite problems should come first, got %s", issues[0].Severity)
	}
}
