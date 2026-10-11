package library

import (
	"strings"
	"testing"

	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/pkg/tmdb"
)

func movie(tmdbID int, title string, year, runtime int) *db.Media {
	m := &db.Media{Type: db.MediaTypeMovie}
	m.TMDbID, m.Title, m.Year, m.Runtime = tmdbID, title, year, runtime
	return m
}

func TestJudgeMatch(t *testing.T) {
	kid1921 := &tmdb.MovieDetails{ID: 2, Title: "The Kid", ReleaseDate: "1921-01-21", Runtime: 68}
	kid2000 := &tmdb.MovieDetails{ID: 1, Title: "The Kid", ReleaseDate: "2000-07-07", Runtime: 104}
	detour1945 := &tmdb.MovieDetails{ID: 4, Title: "Detour", ReleaseDate: "1945-11-30", Runtime: 68}

	tests := []struct {
		name      string
		media     *db.Media
		facts     MovieFileFacts
		suggested *tmdb.MovieDetails
		flagged   bool
		mentions  []string
	}{
		{"matcher agrees with the current match",
			movie(2, "The Kid", 1921, 68), MovieFileFacts{"The Kid", 1921, 68}, kid1921, false, nil},
		{"wrong year and wrong length",
			movie(1, "The Kid", 2000, 104), MovieFileFacts{"The Kid", 1921, 68}, kid1921, true,
			[]string{"1921", "2000", "1h 8m", "1h 44m"}},
		{"unrelated title",
			movie(9, "Avengers: Endgame", 2019, 181), MovieFileFacts{"Charade", 0, 0}, nil, true,
			[]string{`"Charade"`}},
		{"length contradicts and nothing better was found",
			movie(3, "Detour", 2016, 97), MovieFileFacts{"Detour", 0, 68}, nil, true,
			[]string{"1h 8m", "1h 37m"}},
		{"unmatched, and now there is a match",
			movie(0, "Detour", 0, 0), MovieFileFacts{"Detour", 0, 68}, detour1945, true, nil},
		{"unmatched, and still nothing",
			movie(0, "Detour", 0, 0), MovieFileFacts{"Detour", 0, 68}, nil, false, nil},
		{"a different pick with no evidence behind it is not raised",
			movie(1, "The Kid", 2000, 104), MovieFileFacts{"The Kid", 0, 0}, kid1921, false, nil},
		{"nothing confident found, but nothing against the current match either",
			movie(1, "The Kid", 2000, 104), MovieFileFacts{"The Kid", 0, 0}, nil, false, nil},
		{"an extended cut's length doesn't contradict, but the other film fits exactly",
			movie(1, "The Kid", 2000, 80), MovieFileFacts{"The Kid", 0, 68}, kid1921, true,
			[]string{"1h 8m", "1h 20m"}},
		{"one year out is a release-date difference, not a mismatch",
			movie(4, "Detour", 1945, 68), MovieFileFacts{"Detour", 1946, 68}, kid2000, false, nil},
	}

	for _, tt := range tests {
		reason, flagged := judgeMatch(tt.media, tt.facts, tt.suggested)
		if flagged != tt.flagged {
			t.Errorf("%s: flagged = %v, want %v (reason %q)", tt.name, flagged, tt.flagged, reason)
			continue
		}
		if flagged && reason == "" {
			t.Errorf("%s: flagged without a reason", tt.name)
		}
		for _, want := range tt.mentions {
			if !strings.Contains(reason, want) {
				t.Errorf("%s: reason %q does not mention %q", tt.name, reason, want)
			}
		}
	}
}

func TestFormatMinutes(t *testing.T) {
	for minutes, want := range map[int]string{45: "45m", 60: "1h", 68: "1h 8m", 0: "an unknown length"} {
		if got := FormatMinutes(minutes); got != want {
			t.Errorf("FormatMinutes(%d) = %q, want %q", minutes, got, want)
		}
	}
}
