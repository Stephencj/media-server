package library

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stephencjuliano/media-server/internal/db"
)

func TestRelinkMovedFiles(t *testing.T) {
	scanner, database, sources := newMisfiledLibrary(t)
	dir := t.TempDir()
	touch := func(name string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	// Converted: the .mkv is gone and an .mp4 of the same name is there.
	converted := addMovieRow(t, database, sources["movies"], filepath.Join(dir, "Converted.mkv"), "Converted", 100)
	touch("Converted.mp4")
	// Untouched: the file is still where the entry says.
	untouched := addMovieRow(t, database, sources["movies"], touch("Untouched.mkv"), "Untouched", 100)
	touch("Untouched.mp4")
	// Gone, with nothing in its place.
	addMovieRow(t, database, sources["movies"], filepath.Join(dir, "Gone.mkv"), "Gone", 100)
	// Gone, but the only candidate already belongs to another entry.
	addMovieRow(t, database, sources["movies"], filepath.Join(dir, "Shared.mkv"), "Shared", 100)
	addMovieRow(t, database, sources["movies"], touch("Shared.mp4"), "Shared copy", 100)

	probe := func(path string) (*db.MediaFile, error) {
		return &db.MediaFile{FileSize: 1234, Duration: 6000, VideoCodec: "h264", AudioCodec: "aac", Resolution: "1920x1080"}, nil
	}

	plan, err := scanner.relinkMovedFiles(false, probe)
	if err != nil || len(plan) != 1 || plan[0].Title != "Converted" {
		t.Fatalf("plan = %+v, %v; want only Converted", plan, err)
	}
	if got, _ := database.GetMediaByID(converted.ID); got.FilePath != converted.FilePath {
		t.Fatal("planning changed the entry")
	}

	done, err := scanner.relinkMovedFiles(true, probe)
	if err != nil || len(done) != 1 || done[0].Error != "" {
		t.Fatalf("relink = %+v, %v", done, err)
	}
	got, _ := database.GetMediaByID(converted.ID)
	if got.FilePath != filepath.Join(dir, "Converted.mp4") || got.AudioCodec != "aac" || got.FileSize != 1234 {
		t.Errorf("entry after relink = %+v", got.MediaFile)
	}
	if got.Title != "Converted" || got.SourceID != sources["movies"].ID {
		t.Errorf("relink disturbed the rest of the entry: %q, source %d", got.Title, got.SourceID)
	}
	if still, _ := database.GetMediaByID(untouched.ID); still.FilePath != untouched.FilePath {
		t.Error("an entry whose file exists was moved")
	}

	if again, _ := scanner.relinkMovedFiles(true, probe); len(again) != 0 {
		t.Errorf("second run relinked %d more", len(again))
	}
}

func TestRelinkLeavesTheEntryWhenTheNewFileCannotBeRead(t *testing.T) {
	scanner, database, sources := newMisfiledLibrary(t)
	dir := t.TempDir()
	entry := addMovieRow(t, database, sources["movies"], filepath.Join(dir, "Film.mkv"), "Film", 100)
	if err := os.WriteFile(filepath.Join(dir, "Film.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	failing := func(string) (*db.MediaFile, error) { return nil, errors.New("not a video") }
	done, _ := scanner.relinkMovedFiles(true, failing)
	if len(done) != 1 || done[0].Error == "" {
		t.Fatalf("relink = %+v, want one reported failure", done)
	}
	if got, _ := database.GetMediaByID(entry.ID); got.FilePath != entry.FilePath {
		t.Error("the entry was changed although the new file could not be read")
	}
}
