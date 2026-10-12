package library

import (
	"testing"

	"github.com/stephencjuliano/media-server/internal/db"
)

func copyOf(id int64, path string, tmdbID, minutes int, size int64) *db.Media {
	m := &db.Media{ID: id, Type: db.MediaTypeMovie}
	m.FilePath, m.TMDbID, m.Duration, m.FileSize = path, tmdbID, minutes*60, size
	return m
}

func TestGroupMovieCopies(t *testing.T) {
	movies := []*db.Media{
		copyOf(1, "/m/Charade/Charade 1953.m4v", 10, 81, 100),
		copyOf(2, "/m/Charade/Charade 1963.m4v", 11, 116, 100),
		// One film, matched, in two folders; the short rip is not the entry.
		copyOf(3, "/m/Shrek 2.m4v", 20, 59, 300),
		copyOf(4, "/m/Shrek 2 (2004)/Shrek 2.mkv", 20, 93, 200),
		// One film, unmatched, three files in a folder; the largest leads.
		copyOf(5, "/m/Edward Scissor Hands/Edward Scissor Hands.m4v", 0, 104, 100),
		copyOf(6, "/m/Edward Scissor Hands/Edward Scissor Hands HD.m4v", 0, 104, 400),
		copyOf(7, "/m/Edward Scissor Hands/Edward Scissor Hands v1.m4v", 0, 104, 50),
		// Unmatched files that are different films stay apart.
		copyOf(8, "/m/Austin Powers/Austin Powers2.m4v", 0, 95, 100),
		copyOf(9, "/m/Austin Powers/Austin Powers3.m4v", 0, 94, 100),
	}

	entries, copies := GroupMovieCopies(movies)

	var ids []int64
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	want := []int64{1, 2, 4, 6, 8, 9}
	if len(ids) != len(want) {
		t.Fatalf("entries = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("entries = %v, want %v", ids, want)
		}
	}

	if entries[2].CopyCount != 1 || len(copies[4]) != 1 || copies[4][0].ID != 3 {
		t.Errorf("Shrek 2: CopyCount %d, copies %v", entries[2].CopyCount, copies[4])
	}
	if entries[3].CopyCount != 2 || len(copies[6]) != 2 {
		t.Errorf("Edward Scissorhands: CopyCount %d, copies %d", entries[3].CopyCount, len(copies[6]))
	}
	if movies[3].CopyCount != 0 {
		t.Error("GroupMovieCopies changed the rows it was given")
	}

	// Asking from any copy returns the others.
	others := MovieCopies(movies, 7)
	if len(others) != 2 || others[0].ID != 6 || others[1].ID != 5 {
		t.Errorf("copies of 7 = %v", others)
	}
	if others := MovieCopies(movies, 1); len(others) != 0 {
		t.Errorf("Charade 1953 has copies: %v", others)
	}
}
