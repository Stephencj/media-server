package library

import (
	"testing"

	"github.com/stephencjuliano/media-server/internal/db"
)

// Extras scanned before their film was in the library, or in a folder named a
// little differently from it, are linked on the next scan.
func TestScanLinksOrphanedExtrasToTheirFilm(t *testing.T) {
	scanner, database, sources := newMisfiledLibrary(t)
	features, err := database.CreateMediaSource(&db.MediaSource{
		Name: "features", Path: "/media/Movie Special Features", Type: "local", Enabled: true,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	orphan := func(path string) *db.Extra {
		extra := &db.Extra{Title: "clip", Category: db.ExtraCategoryOther}
		extra.SourceID = features.ID
		extra.FilePath = path
		created, err := database.CreateExtra(extra)
		if err != nil {
			t.Fatalf("create extra: %v", err)
		}
		return created
	}
	ecto := orphan("/media/Movie Special Features/Ghost Busters/Ghostbusters Ecto1.m4v")
	bloopers := orphan("/media/Movie Special Features/Scott Pilgrim vs the World/Scott Pilgrim Bloopers.m4v")
	shaun := orphan("/media/Movie Special Features/Shaun of the Dead/Raw Meat.m4v")
	nothing := orphan("/media/Movie Special Features/Aladdin (1992)/Aladdin - Trailer.m4v")

	movies := sources["movies"]
	ghostbusters := addMovieRow(t, database, movies, "/media/Movies/Ghost Busters/Ghostbusters.m4v", "Ghostbusters", 105)
	pilgrim := addMovieRow(t, database, movies, "/media/Movies/Scott Pilgrim vs the World/Scott Pilgrim.m4v", "Scott Pilgrim vs. the World", 112)
	shaunFilm := addMovieRow(t, database, movies, "/media/Movies/Shaun Of The Dead/Shaun Of The Dead.m4v", "Shaun of the Dead", 99)

	for _, extra := range []*db.Extra{ecto, bloopers, shaun, nothing} {
		if err := scanner.processExtraFile(extra.FilePath, features); err != nil {
			t.Fatalf("scan %s: %v", extra.FilePath, err)
		}
	}

	parent := func(extra *db.Extra) *int64 {
		got, err := database.GetExtraByID(extra.ID)
		if err != nil {
			t.Fatalf("read extra: %v", err)
		}
		return got.MovieID
	}
	for _, c := range []struct {
		name  string
		extra *db.Extra
		want  int64
	}{
		{"Ghost Busters is Ghostbusters", ecto, ghostbusters.ID},
		{"the folder's name is the film's folder", bloopers, pilgrim.ID},
		{"a film that arrived after its extras", shaun, shaunFilm.ID},
	} {
		if got := parent(c.extra); got == nil || *got != c.want {
			t.Errorf("%s: linked to %v, want %d", c.name, got, c.want)
		}
	}
	if got := parent(nothing); got != nil {
		t.Errorf("no such film: linked to %d", *got)
	}
}

func TestFilmNamedLeavesTwoFilmsOfOneNameAlone(t *testing.T) {
	scanner, database, sources := newMisfiledLibrary(t)
	movies := sources["movies"]
	for _, f := range []struct {
		path string
		tmdb int
	}{{"/media/Movies/Hamlet 1948/Hamlet.m4v", 1}, {"/media/Movies/Hamlet 1996/Hamlet.m4v", 2}} {
		media := &db.Media{Type: db.MediaTypeMovie}
		media.SourceID, media.FilePath, media.Title, media.TMDbID, media.Duration = movies.ID, f.path, "Hamlet", f.tmdb, 6000
		if _, err := database.CreateMedia(media); err != nil {
			t.Fatalf("create media: %v", err)
		}
	}
	if film := scanner.filmNamed("Hamlet"); film != nil {
		t.Errorf("two different Hamlets: picked %s", film.FilePath)
	}

	// Two copies of one film are one film: the longer copy.
	for i, path := range []string{"/media/Movies/Charade/Charade.m4v", "/media/Movies/Charade/Charade HD.m4v"} {
		media := &db.Media{Type: db.MediaTypeMovie}
		media.SourceID, media.FilePath, media.Title, media.TMDbID, media.Duration = movies.ID, path, "Charade", 7, 6000+i
		if _, err := database.CreateMedia(media); err != nil {
			t.Fatalf("create media: %v", err)
		}
	}
	if film := scanner.filmNamed("charade"); film == nil || film.FilePath != "/media/Movies/Charade/Charade HD.m4v" {
		t.Errorf("two copies of Charade: got %v", film)
	}
}
