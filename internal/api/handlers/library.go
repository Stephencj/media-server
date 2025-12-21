package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/internal/library"
)

type LibraryHandler struct {
	db      *db.DB
	cfg     *config.Config
	scanner *library.Scanner
}

func NewLibraryHandler(database *db.DB, cfg *config.Config) *LibraryHandler {
	return &LibraryHandler{
		db:      database,
		cfg:     cfg,
		scanner: library.NewScanner(database, cfg),
	}
}

type PaginatedResponse struct {
	Items  interface{} `json:"items"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

// GetMovies returns all movies in the library
func (h *LibraryHandler) GetMovies(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	movies, err := h.db.GetMediaByType(db.MediaTypeMovie, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movies"})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Items:  movies,
		Limit:  limit,
		Offset: offset,
	})
}

// GetShows returns all TV shows in the library
func (h *LibraryHandler) GetShows(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	shows, err := h.db.GetMediaByType(db.MediaTypeTVShow, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch TV shows"})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Items:  shows,
		Limit:  limit,
		Offset: offset,
	})
}

// GetRecent returns recently added media
func (h *LibraryHandler) GetRecent(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if limit > 50 {
		limit = 50
	}

	media, err := h.db.GetRecentMedia(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recent media"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": media})
}

// GetMedia returns a single media item by ID
func (h *LibraryHandler) GetMedia(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	media, err := h.db.GetMediaByID(id)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
		return
	}

	c.JSON(http.StatusOK, media)
}

// TriggerScan initiates a library scan
func (h *LibraryHandler) TriggerScan(c *gin.Context) {
	if h.scanner.IsRunning() {
		c.JSON(http.StatusConflict, gin.H{
			"message": "Scan already in progress",
			"status":  "scanning",
		})
		return
	}

	// Run scan asynchronously
	go func() {
		if err := h.scanner.ScanAll(); err != nil {
			// Log error but don't fail - scan is async
			println("Scan error:", err.Error())
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Library scan started",
		"status":  "scanning",
	})
}
