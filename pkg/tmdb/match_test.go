package tmdb

import (
	"errors"
	"testing"
)

// catalogue is a stand-in for TMDB: search results plus the running times
// only the detail record carries.
type catalogue []struct {
	id         int
	title      string
	date       string
	popularity float64
	runtime    int
}

func (c catalogue) results() []MovieSearchResult {
	out := make([]MovieSearchResult, len(c))
	for i, m := range c {
		out[i] = MovieSearchResult{ID: m.id, Title: m.title, ReleaseDate: m.date, Popularity: m.popularity}
	}
	return out
}

func (c catalogue) fetch(id int) (*MovieDetails, error) {
	for _, m := range c {
		if m.id == id {
			return &MovieDetails{ID: m.id, Title: m.title, ReleaseDate: m.date, Runtime: m.runtime}, nil
		}
	}
	return nil, errors.New("not found")
}

func TestPickMovie(t *testing.T) {
	kids := catalogue{
		{1, "The Kid", "2000-07-07", 40, 104},
		{2, "The Kid", "1921-01-21", 12, 68},
		{3, "The Kid", "2019-03-08", 15, 100},
		{4, "The Kid Brother", "1927-01-22", 3, 84},
	}
	detours := catalogue{
		{1, "Detour", "2016-11-26", 25, 97},
		{2, "Detour", "1945-11-30", 8, 68},
	}
	runners := catalogue{
		{1, "Blade Runner", "1982-06-25", 60, 117},
		{2, "Blade Runner 2049", "2017-10-04", 70, 164},
	}
	unrelated := catalogue{
		{1, "Avengers: Endgame", "2019-04-24", 200, 181},
		{2, "Joker", "2019-10-01", 150, 122},
	}
	subtitled := catalogue{
		{1, "Birdman or (The Unexpected Virtue of Ignorance)", "2014-10-17", 30, 120},
	}
	aliens := catalogue{
		{1, "Alien³", "1992-05-22", 30, 114},
		{2, "Aliens", "1986-07-18", 50, 137},
		{3, "Alien: Romulus", "2024-08-13", 90, 119},
	}

	tests := []struct {
		name    string
		cat     catalogue
		title   string
		year    int
		runtime int
		want    int // matched id, 0 for no match
	}{
		{"year separates same-named films", kids, "The Kid", 1921, 68, 2},
		{"year alone is enough", kids, "The Kid", 1921, 0, 2},
		{"running time separates them when the name has no year", kids, "The Kid", 0, 68, 2},
		{"popularity only breaks a tie", kids, "The Kid", 0, 0, 1},
		{"running time beats popularity", detours, "Detour", 0, 68, 2},
		{"a number in the title is part of the title", runners, "Blade Runner 2049", 2017, 164, 2},
		{"and its absence is too", runners, "Blade Runner", 1982, 117, 1},
		{"the right year cannot rescue an unrelated title", unrelated, "Charade", 2019, 0, 0},
		{"nor can popularity", unrelated, "Charade", 0, 0, 0},
		{"a subtitle on TMDB's side still matches", subtitled, "Birdman", 2014, 120, 1},
		{"a sequel is not a subtitle", aliens, "Alien", 1979, 117, 0},
		{"same title, year far out, nothing else to go on", detours, "Detour", 1975, 0, 0},
		{"a wrong year in the name is forgiven when the running time agrees", detours, "Detour", 1946, 68, 2},
		{"a matching name with the wrong length is not the film", catalogue{{1, "Detour", "2016-11-26", 25, 97}}, "Detour", 0, 68, 0},
		{"no results", nil, "Charade", 1963, 113, 0},
	}

	for _, tt := range tests {
		got, err := pickMovie(tt.cat.results(), tt.title, tt.year, tt.runtime, tt.cat.fetch)
		if err != nil {
			t.Errorf("%s: unexpected error %v", tt.name, err)
			continue
		}
		id := 0
		if got != nil {
			id = got.ID
		}
		if id != tt.want {
			t.Errorf("%s: matched id %d, want %d", tt.name, id, tt.want)
		}
	}
}

func TestPickMovieSurvivesAFailedLookup(t *testing.T) {
	cat := catalogue{
		{1, "Detour", "2016-11-26", 25, 97},
		{2, "Detour", "1945-11-30", 8, 68},
	}
	fetch := func(id int) (*MovieDetails, error) {
		if id == 1 {
			return nil, errors.New("timeout")
		}
		return cat.fetch(id)
	}

	got, err := pickMovie(cat.results(), "Detour", 0, 68, fetch)
	if err != nil || got == nil || got.ID != 2 {
		t.Errorf("got %v, %v; want the 1945 film", got, err)
	}

	failing := func(int) (*MovieDetails, error) { return nil, errors.New("timeout") }
	if got, err := pickMovie(cat.results(), "Detour", 0, 68, failing); got != nil || err == nil {
		t.Errorf("got %v, %v; want the lookup error", got, err)
	}
}

func TestMovieTitleSimilarity(t *testing.T) {
	tests := []struct {
		a, b string
		want float64
	}{
		{"The Kid", "Kid, The", 1.0},
		{"Amelie", "Amélie", 1.0},
		{"Dont Look Now", "Don't Look Now", 1.0},
		{"Fast and Furious", "Fast & Furious", 1.0},
		{"Rocky 2", "Rocky II", 1.0},
		{"Spider Man", "Spider-Man", 1.0},
		{"Mission Impossible Fallout", "Mission: Impossible - Fallout", 1.0},
		{"Birdman", "Birdman or (The Unexpected Virtue of Ignorance)", 0.75},
		{"Alien", "Alien 3", 0.5},
		{"Blade Runner", "Blade Runner 2049", 2.0 / 3.0},
		{"Night of Living Dead", "Night of the Living Dead", 1.0},
		{"Charade", "Avengers: Endgame", 0},
		{"", "Charade", 0},
	}

	for _, tt := range tests {
		if got := movieTitleSimilarity(tt.a, tt.b); got < tt.want-0.001 || got > tt.want+0.001 {
			t.Errorf("movieTitleSimilarity(%q, %q) = %.3f, want %.3f", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestRuntimeScore(t *testing.T) {
	tests := []struct {
		file, tmdb int
		want       float64
	}{
		{68, 68, 15},
		{113, 115, 15},
		{200, 205, 15}, // within 3% of a long film
		{144, 129, 0},  // an extended cut
		{68, 104, -25}, // a different film
		{0, 104, 0},    // file length unknown
		{68, 0, 0},     // TMDB has none recorded
	}

	for _, tt := range tests {
		if got := runtimeScore(tt.file, tt.tmdb); got != tt.want {
			t.Errorf("runtimeScore(%d, %d) = %v, want %v", tt.file, tt.tmdb, got, tt.want)
		}
	}
}
