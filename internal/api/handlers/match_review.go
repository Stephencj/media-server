package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/internal/library"
	"github.com/stephencjuliano/media-server/pkg/tmdb"
)

// MatchReviewHandler runs the "check matches" queue: it re-matches movies
// already in the library and holds the ones that look wrong for a person to
// accept, replace or keep. Checking never changes a movie; only the accept,
// accept-all and choose endpoints do.
type MatchReviewHandler struct {
	db   *db.DB
	tmdb *tmdb.Client

	mu  sync.Mutex
	run matchCheckRun
}

type matchCheckRun struct {
	Running    bool       `json:"running"`
	Checked    int        `json:"checked"`
	Total      int        `json:"total"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// matchFilm is one side of a comparison: the film a movie is shown as, the
// film suggested for it, or a candidate to choose.
type matchFilm struct {
	TMDbID     int    `json:"tmdb_id"`
	Title      string `json:"title"`
	Year       int    `json:"year"`
	Runtime    int    `json:"runtime"` // minutes
	PosterPath string `json:"poster_path,omitempty"`
}

type matchReviewItem struct {
	MediaID     int64  `json:"media_id"`
	FileName    string `json:"file_name"`
	FileTitle   string `json:"file_title"`
	FileYear    int    `json:"file_year"`
	FileRuntime int    `json:"file_runtime"` // minutes
	// Current is null when the movie has no match yet.
	Current *matchFilm `json:"current"`
	// Suggested is null when the suggestion is to drop the match and show
	// the file's own name.
	Suggested *matchFilm `json:"suggested"`
	Reason    string     `json:"reason"`
}

type matchCandidate struct {
	matchFilm
	Overview  string `json:"overview,omitempty"`
	Fit       string `json:"fit"`
	IsCurrent bool   `json:"is_current"`
	IsBestFit bool   `json:"is_best_fit"`
}

// How many search results get their running time looked up for the picker.
const maxMatchCandidates = 8

// Pause between movies during a check, to stay well inside TMDB's rate limit.
const matchCheckPause = 200 * time.Millisecond

func NewMatchReviewHandler(database *db.DB, cfg *config.Config) *MatchReviewHandler {
	return &MatchReviewHandler{
		db:   database,
		tmdb: tmdb.NewClient(cfg.TMDbAPIKey),
	}
}

// GET /api/library/match-review
// The queue, plus the state of the check that fills it.
func (h *MatchReviewHandler) List(c *gin.Context) {
	reviews, err := h.db.ListPendingMatchReviews()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load match reviews"})
		return
	}

	items := make([]matchReviewItem, 0, len(reviews))
	for _, review := range reviews {
		media, err := h.db.GetMediaByID(review.MediaID)
		if err != nil {
			continue
		}
		items = append(items, buildReviewItem(media, review))
	}

	c.JSON(http.StatusOK, gin.H{"status": h.status(), "items": items})
}

// POST /api/library/match-review/scan
// Starts checking every movie in the background.
func (h *MatchReviewHandler) Scan(c *gin.Context) {
	if !h.tmdb.IsConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TMDB API key not configured"})
		return
	}

	h.mu.Lock()
	if h.run.Running {
		h.mu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"status": h.status()})
		return
	}
	h.run = matchCheckRun{Running: true}
	h.mu.Unlock()

	go h.checkAll()

	c.JSON(http.StatusAccepted, gin.H{"status": h.status()})
}

func (h *MatchReviewHandler) status() matchCheckRun {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.run
}

func (h *MatchReviewHandler) checkAll() {
	finish := func(errText string) {
		now := time.Now()
		h.mu.Lock()
		h.run.Running = false
		h.run.FinishedAt = &now
		if errText != "" {
			h.run.Error = errText
		}
		h.mu.Unlock()
	}

	movies, err := h.db.GetMediaByType(db.MediaTypeMovie, 1_000_000, 0)
	if err != nil {
		finish("Could not read the library")
		return
	}

	h.mu.Lock()
	h.run.Total = len(movies)
	h.mu.Unlock()

	failures := 0
	for _, media := range movies {
		if err := h.checkOne(media); err != nil {
			failures++
			log.Printf("Match check failed for %s: %v", media.FilePath, err)
		}
		h.mu.Lock()
		h.run.Checked++
		h.mu.Unlock()
		time.Sleep(matchCheckPause)
	}

	if failures > 0 {
		finish(fmt.Sprintf("%d of %d movies could not be checked", failures, len(movies)))
		return
	}
	finish("")
}

// checkOne re-matches a movie and records the result. It never touches the
// movie itself.
func (h *MatchReviewHandler) checkOne(media *db.Media) error {
	suggested, reason, flagged, err := library.ReviewMovieMatch(h.tmdb, media)
	if err != nil {
		return err
	}
	if !flagged {
		return h.db.DeleteMatchReview(media.ID)
	}

	review := &db.MatchReview{
		MediaID:       media.ID,
		Status:        db.MatchReviewPending,
		CurrentTMDbID: media.TMDbID,
		Reason:        reason,
	}
	if suggested != nil {
		encoded, err := json.Marshal(suggested)
		if err != nil {
			return err
		}
		review.SuggestedTMDbID = suggested.ID
		review.SuggestedJSON = string(encoded)
	}

	// Don't ask again about a question that was already answered "keep".
	if existing, err := h.db.GetMatchReview(media.ID); err == nil &&
		existing.Status == db.MatchReviewKept &&
		existing.CurrentTMDbID == review.CurrentTMDbID &&
		existing.SuggestedTMDbID == review.SuggestedTMDbID {
		return nil
	}

	return h.db.UpsertMatchReview(review)
}

// POST /api/library/match-review/:mediaId/accept
// Applies the suggestion for one movie.
func (h *MatchReviewHandler) Accept(c *gin.Context) {
	media, review, ok := h.loadReview(c)
	if !ok {
		return
	}
	if err := h.applySuggestion(media, review); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply the suggestion"})
		return
	}
	c.JSON(http.StatusOK, media)
}

// POST /api/library/match-review/accept-all
// Applies every pending suggestion.
func (h *MatchReviewHandler) AcceptAll(c *gin.Context) {
	reviews, err := h.db.ListPendingMatchReviews()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load match reviews"})
		return
	}

	accepted := 0
	for _, review := range reviews {
		media, err := h.db.GetMediaByID(review.MediaID)
		if err != nil {
			continue
		}
		if err := h.applySuggestion(media, review); err != nil {
			log.Printf("Could not apply suggested match for %s: %v", media.FilePath, err)
			continue
		}
		accepted++
	}

	c.JSON(http.StatusOK, gin.H{"accepted": accepted, "failed": len(reviews) - accepted})
}

// POST /api/library/match-review/:mediaId/keep
// Confirms the current match; the movie won't be raised again for this
// suggestion.
func (h *MatchReviewHandler) Keep(c *gin.Context) {
	media, _, ok := h.loadReview(c)
	if !ok {
		return
	}
	if err := h.db.SetMatchReviewStatus(media.ID, db.MatchReviewKept); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the review"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /api/library/match-review/:mediaId/candidates?query=
// Films a movie could be, each with how well it fits the file. query defaults
// to the title in the file's name.
func (h *MatchReviewHandler) Candidates(c *gin.Context) {
	media, ok := h.loadMovie(c)
	if !ok {
		return
	}
	if !h.tmdb.IsConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TMDB API key not configured"})
		return
	}

	facts := library.FactsForMovie(media)
	query := strings.TrimSpace(c.Query("query"))
	if query == "" {
		query = facts.Title
	}

	// No year filter: the point of the picker is to see the alternatives.
	results, err := h.tmdb.SearchMovieWithResults(query, 0)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "TMDB search failed"})
		return
	}
	if len(results) > maxMatchCandidates {
		results = results[:maxMatchCandidates]
	}

	type ranked struct {
		candidate matchCandidate
		rank      int
	}
	var all []ranked
	for i := range results {
		details, err := h.tmdb.GetMovieDetails(results[i].ID)
		if err != nil {
			continue
		}
		film := filmFromDetails(details)
		all = append(all, ranked{
			candidate: matchCandidate{
				matchFilm: film,
				Overview:  details.Overview,
				Fit:       describeFit(facts, film),
				IsCurrent: media.TMDbID != 0 && details.ID == media.TMDbID,
			},
			rank: fitRank(facts, film),
		})
	}

	sort.SliceStable(all, func(a, b int) bool { return all[a].rank > all[b].rank })

	candidates := make([]matchCandidate, 0, len(all))
	for i, r := range all {
		// Only call something the best fit when the file backs it up.
		r.candidate.IsBestFit = i == 0 && r.rank >= bestFitRank
		candidates = append(candidates, r.candidate)
	}

	c.JSON(http.StatusOK, gin.H{
		"query":        query,
		"file_name":    filepath.Base(media.FilePath),
		"file_year":    facts.Year,
		"file_runtime": facts.Runtime,
		"candidates":   candidates,
	})
}

// POST /api/library/match-review/:mediaId/choose  {"tmdb_id": 123}
// Sets a movie's match by hand and clears any pending review for it.
func (h *MatchReviewHandler) Choose(c *gin.Context) {
	media, ok := h.loadMovie(c)
	if !ok {
		return
	}

	var req struct {
		TMDbID int `json:"tmdb_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	details, err := h.tmdb.GetMovieDetails(req.TMDbID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch movie details"})
		return
	}

	applyMovieDetails(media, details)
	if err := h.db.UpdateMedia(media); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update media"})
		return
	}
	h.db.DeleteMatchReview(media.ID)

	c.JSON(http.StatusOK, media)
}

func (h *MatchReviewHandler) loadMovie(c *gin.Context) (*db.Media, bool) {
	mediaID, err := strconv.ParseInt(c.Param("mediaId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return nil, false
	}
	media, err := h.db.GetMediaByID(mediaID)
	if err != nil || media.Type != db.MediaTypeMovie {
		c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
		return nil, false
	}
	return media, true
}

func (h *MatchReviewHandler) loadReview(c *gin.Context) (*db.Media, *db.MatchReview, bool) {
	media, ok := h.loadMovie(c)
	if !ok {
		return nil, nil, false
	}
	review, err := h.db.GetMatchReview(media.ID)
	if err != nil || review.Status != db.MatchReviewPending {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nothing to review for this movie"})
		return nil, nil, false
	}
	return media, review, true
}

// applySuggestion makes a review's suggestion the movie's match. It works
// from the details stored when the movie was checked, so accepting a whole
// queue makes no network calls.
func (h *MatchReviewHandler) applySuggestion(media *db.Media, review *db.MatchReview) error {
	if review.SuggestedTMDbID == 0 {
		clearMovieMatch(media)
	} else {
		var details tmdb.MovieDetails
		if err := json.Unmarshal([]byte(review.SuggestedJSON), &details); err != nil {
			return err
		}
		applyMovieDetails(media, &details)
	}

	if err := h.db.UpdateMedia(media); err != nil {
		return err
	}
	return h.db.DeleteMatchReview(media.ID)
}

// clearMovieMatch returns a movie to what its file says, for when no film is
// a confident match.
func clearMovieMatch(media *db.Media) {
	facts := library.FactsForMovie(media)
	media.Title = facts.Title
	media.Year = facts.Year
	media.OriginalTitle = ""
	media.Overview = ""
	media.PosterPath = ""
	media.BackdropPath = ""
	media.Rating = 0
	media.Runtime = 0
	media.Genres = ""
	media.TMDbID = 0
	media.IMDbID = ""
}

func buildReviewItem(media *db.Media, review *db.MatchReview) matchReviewItem {
	facts := library.FactsForMovie(media)
	item := matchReviewItem{
		MediaID:     media.ID,
		FileName:    filepath.Base(media.FilePath),
		FileTitle:   facts.Title,
		FileYear:    facts.Year,
		FileRuntime: facts.Runtime,
		Reason:      review.Reason,
	}
	if media.TMDbID != 0 {
		item.Current = &matchFilm{
			TMDbID: media.TMDbID, Title: media.Title, Year: media.Year,
			Runtime: media.Runtime, PosterPath: media.PosterPath,
		}
	}
	if review.SuggestedTMDbID != 0 {
		var details tmdb.MovieDetails
		if err := json.Unmarshal([]byte(review.SuggestedJSON), &details); err == nil {
			film := filmFromDetails(&details)
			item.Suggested = &film
		}
	}
	return item
}

func filmFromDetails(details *tmdb.MovieDetails) matchFilm {
	return matchFilm{
		TMDbID: details.ID, Title: details.Title, Year: extractYear(details.ReleaseDate),
		Runtime: details.Runtime, PosterPath: details.PosterPath,
	}
}

// bestFitRank is the least a candidate needs to be called the best fit: its
// running time agreeing with the file's, or the year when there is no running
// time to go on.
const bestFitRank = 4

// fitRank orders candidates for the picker by how well they fit the file.
func fitRank(facts library.MovieFileFacts, film matchFilm) int {
	rank := 0
	if tmdb.TitlesAgree(facts.Title, film.Title) {
		rank += 2
	}
	if facts.Year > 0 && film.Year > 0 {
		switch diff := facts.Year - film.Year; {
		case diff == 0:
			rank += 2
		case diff == 1 || diff == -1:
			rank++
		default:
			rank -= 2
		}
	}
	switch {
	case tmdb.RuntimesAgree(facts.Runtime, film.Runtime):
		rank += 3
	case tmdb.RuntimesConflict(facts.Runtime, film.Runtime):
		rank -= 3
	}
	return rank
}

// describeFit says how a candidate compares with the file: "Same year, same
// running time", "79 years off, 36 min longer".
func describeFit(facts library.MovieFileFacts, film matchFilm) string {
	var parts []string

	if facts.Year > 0 && film.Year > 0 {
		switch diff := facts.Year - film.Year; {
		case diff == 0:
			parts = append(parts, "same year")
		case diff == 1 || diff == -1:
			parts = append(parts, "1 year off")
		default:
			if diff < 0 {
				diff = -diff
			}
			parts = append(parts, fmt.Sprintf("%d years off", diff))
		}
	}

	if facts.Runtime > 0 && film.Runtime > 0 {
		switch diff := film.Runtime - facts.Runtime; {
		case tmdb.RuntimesAgree(facts.Runtime, film.Runtime):
			parts = append(parts, "same running time")
		case diff > 0:
			parts = append(parts, fmt.Sprintf("%d min longer", diff))
		default:
			parts = append(parts, fmt.Sprintf("%d min shorter", -diff))
		}
	}

	if len(parts) == 0 {
		return ""
	}
	text := strings.Join(parts, ", ")
	return strings.ToUpper(text[:1]) + text[1:]
}
