package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/internal/library"
	"github.com/stephencjuliano/media-server/pkg/tmdb"
)

// matchReviewFixture is a library of one mismatched movie with a suggestion
// already waiting, which is everything the queue endpoints need: none of them
// call TMDB.
type matchReviewFixture struct {
	db     *db.DB
	router *gin.Engine
	media  *db.Media
}

func newMatchReviewFixture(t *testing.T, suggested *tmdb.MovieDetails) *matchReviewFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	database, err := db.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	source, err := database.CreateMediaSource(&db.MediaSource{Name: "Movies", Path: "/movies", Type: "local", Enabled: true})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	media := &db.Media{Type: db.MediaTypeMovie}
	media.SourceID = source.ID
	media.FilePath = "/movies/The.Kid.1921.1080p.mkv"
	media.Duration = 68 * 60
	media.Title, media.Year, media.Runtime, media.TMDbID = "The Kid", 2000, 104, 1
	media.PosterPath, media.Overview = "/wrong.jpg", "A man meets his younger self."
	media, err = database.CreateMedia(media)
	if err != nil {
		t.Fatalf("create media: %v", err)
	}

	review := &db.MatchReview{
		MediaID: media.ID, Status: db.MatchReviewPending, CurrentTMDbID: 1,
		Reason: "The file says 1921, the current match is from 2000.",
	}
	if suggested != nil {
		encoded, _ := json.Marshal(suggested)
		review.SuggestedTMDbID, review.SuggestedJSON = suggested.ID, string(encoded)
	}
	if err := database.UpsertMatchReview(review); err != nil {
		t.Fatalf("seed review: %v", err)
	}

	h := NewMatchReviewHandler(database, &config.Config{})
	r := gin.New()
	g := r.Group("/api/library")
	g.GET("/match-review", h.List)
	g.POST("/match-review/scan", h.Scan)
	g.POST("/match-review/accept-all", h.AcceptAll)
	g.POST("/match-review/:mediaId/accept", h.Accept)
	g.POST("/match-review/:mediaId/keep", h.Keep)

	return &matchReviewFixture{db: database, router: r, media: media}
}

func (f *matchReviewFixture) do(method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func (f *matchReviewFixture) queue(t *testing.T) []matchReviewItem {
	t.Helper()
	w := f.do(http.MethodGet, "/api/library/match-review")
	var body struct {
		Items []matchReviewItem `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); w.Code != http.StatusOK || err != nil {
		t.Fatalf("list -> %d %v: %s", w.Code, err, w.Body.String())
	}
	return body.Items
}

var kid1921 = &tmdb.MovieDetails{
	ID: 2, Title: "The Kid", ReleaseDate: "1921-01-21", Runtime: 68,
	PosterPath: "/right.jpg", Overview: "A tramp raises an abandoned child.",
}

func TestMatchReviewListDescribesBothSides(t *testing.T) {
	f := newMatchReviewFixture(t, kid1921)

	items := f.queue(t)
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	item := items[0]
	if item.FileName != "The.Kid.1921.1080p.mkv" || item.FileTitle != "The Kid" || item.FileYear != 1921 || item.FileRuntime != 68 {
		t.Errorf("file facts = %+v", item)
	}
	if item.Current == nil || item.Current.Year != 2000 || item.Current.Runtime != 104 {
		t.Errorf("current = %+v", item.Current)
	}
	if item.Suggested == nil || item.Suggested.TMDbID != 2 || item.Suggested.Year != 1921 {
		t.Errorf("suggested = %+v", item.Suggested)
	}
}

func TestMatchReviewAcceptAppliesTheSuggestion(t *testing.T) {
	f := newMatchReviewFixture(t, kid1921)

	path := "/api/library/match-review/" + itoa(f.media.ID) + "/accept"
	if w := f.do(http.MethodPost, path); w.Code != http.StatusOK {
		t.Fatalf("accept -> %d: %s", w.Code, w.Body.String())
	}

	got, _ := f.db.GetMediaByID(f.media.ID)
	if got.TMDbID != 2 || got.Year != 1921 || got.Runtime != 68 || got.PosterPath != "/right.jpg" {
		t.Errorf("movie after accept = %+v", got.TMDBMetadata)
	}
	if items := f.queue(t); len(items) != 0 {
		t.Errorf("queue still has %d items", len(items))
	}
	// A resolved review can't be accepted twice.
	if w := f.do(http.MethodPost, path); w.Code != http.StatusNotFound {
		t.Errorf("second accept -> %d, want 404", w.Code)
	}
}

func TestMatchReviewAcceptWithNoSuggestionFallsBackToTheFileName(t *testing.T) {
	f := newMatchReviewFixture(t, nil)

	if items := f.queue(t); len(items) != 1 || items[0].Suggested != nil {
		t.Fatalf("queue = %+v, want one item with no suggested film", items)
	}
	if w := f.do(http.MethodPost, "/api/library/match-review/"+itoa(f.media.ID)+"/accept"); w.Code != http.StatusOK {
		t.Fatalf("accept -> %d: %s", w.Code, w.Body.String())
	}

	got, _ := f.db.GetMediaByID(f.media.ID)
	if got.TMDbID != 0 || got.Title != "The Kid" || got.Year != 1921 || got.PosterPath != "" || got.Overview != "" {
		t.Errorf("movie after accept = %+v", got.TMDBMetadata)
	}
}

func TestMatchReviewKeepLeavesTheMovieAloneAndStaysQuiet(t *testing.T) {
	f := newMatchReviewFixture(t, kid1921)

	if w := f.do(http.MethodPost, "/api/library/match-review/"+itoa(f.media.ID)+"/keep"); w.Code != http.StatusNoContent {
		t.Fatalf("keep -> %d: %s", w.Code, w.Body.String())
	}

	got, _ := f.db.GetMediaByID(f.media.ID)
	if got.TMDbID != 1 || got.Year != 2000 {
		t.Errorf("keep changed the movie: %+v", got.TMDBMetadata)
	}
	if items := f.queue(t); len(items) != 0 {
		t.Errorf("queue still has %d items", len(items))
	}
	review, err := f.db.GetMatchReview(f.media.ID)
	if err != nil || review.Status != db.MatchReviewKept {
		t.Errorf("review after keep = %+v, %v; the kept answer has to be remembered", review, err)
	}
}

func TestMatchReviewAcceptAll(t *testing.T) {
	f := newMatchReviewFixture(t, kid1921)

	w := f.do(http.MethodPost, "/api/library/match-review/accept-all")
	var body struct{ Accepted, Failed int }
	json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != http.StatusOK || body.Accepted != 1 || body.Failed != 0 {
		t.Fatalf("accept-all -> %d %s", w.Code, w.Body.String())
	}
	if got, _ := f.db.GetMediaByID(f.media.ID); got.TMDbID != 2 {
		t.Errorf("movie not updated: %+v", got.TMDBMetadata)
	}
}

func TestMatchReviewScanNeedsTMDB(t *testing.T) {
	f := newMatchReviewFixture(t, kid1921)
	if w := f.do(http.MethodPost, "/api/library/match-review/scan"); w.Code != http.StatusBadRequest {
		t.Errorf("scan without an API key -> %d, want 400", w.Code)
	}
}

func TestDescribeFit(t *testing.T) {
	facts := library.MovieFileFacts{Title: "The Kid", Year: 1921, Runtime: 68}
	tests := []struct {
		film matchFilm
		want string
	}{
		{matchFilm{Year: 1921, Runtime: 68}, "Same year, same running time"},
		{matchFilm{Year: 2000, Runtime: 104}, "79 years off, 36 min longer"},
		{matchFilm{Year: 1922, Runtime: 50}, "1 year off, 18 min shorter"},
		{matchFilm{Year: 0, Runtime: 0}, ""},
	}
	for _, tt := range tests {
		if got := describeFit(facts, tt.film); got != tt.want {
			t.Errorf("describeFit(%+v) = %q, want %q", tt.film, got, tt.want)
		}
	}

	right := fitRank(facts, matchFilm{Title: "The Kid", Year: 1921, Runtime: 68})
	wrong := fitRank(facts, matchFilm{Title: "The Kid", Year: 2000, Runtime: 104})
	if right < bestFitRank || wrong >= bestFitRank || right <= wrong {
		t.Errorf("fitRank right = %d, wrong = %d, bar = %d", right, wrong, bestFitRank)
	}
}

func itoa(id int64) string {
	b, _ := json.Marshal(id)
	return string(b)
}
