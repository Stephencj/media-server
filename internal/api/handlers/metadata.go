// Router setup (add to internal/api/router.go):
//
// metadataHandler := handlers.NewMetadataHandler(database, cfg)
// protected.POST("/media/:id/metadata/search", metadataHandler.SearchTMDB)
// protected.PUT("/media/:id/metadata/apply", metadataHandler.ApplyMetadata)
// protected.POST("/media/:id/metadata/refresh", metadataHandler.RefreshMetadata)

package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/pkg/tmdb"
)

type MetadataHandler struct {
	db   *db.DB
	tmdb *tmdb.Client
}

func NewMetadataHandler(database *db.DB, cfg *config.Config) *MetadataHandler {
	return &MetadataHandler{
		db:   database,
		tmdb: tmdb.NewClient(cfg.TMDbAPIKey),
	}
}

// POST /api/media/:id/metadata/search
// Search TMDB and return multiple results for user to choose from
func (h *MetadataHandler) SearchTMDB(c *gin.Context) {
	mediaID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
		Year  int    `json:"year"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get media to determine type
	media, err := h.db.GetMediaByID(mediaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	// Search TMDB based on media type
	if media.Type == db.MediaTypeMovie {
		results, err := h.tmdb.SearchMovieWithResults(req.Title, req.Year)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "TMDB search failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"results": results})
	} else if media.Type == db.MediaTypeTVShow {
		results, err := h.tmdb.SearchTVWithResults(req.Title, req.Year)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "TMDB search failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"results": results})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media type"})
	}
}

// PUT /api/media/:id/metadata/apply
// Apply metadata from a specific TMDB ID chosen by user
func (h *MetadataHandler) ApplyMetadata(c *gin.Context) {
	mediaID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	var req struct {
		TMDbID int `json:"tmdb_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing media
	media, err := h.db.GetMediaByID(mediaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	// Fetch metadata from TMDB
	if media.Type == db.MediaTypeMovie {
		details, err := h.tmdb.GetMovieDetails(req.TMDbID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch movie details"})
			return
		}

		// Apply metadata
		h.applyMovieMetadata(media, details)
	} else if media.Type == db.MediaTypeTVShow {
		details, err := h.tmdb.GetTVDetails(req.TMDbID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch TV show details"})
			return
		}

		// Apply metadata
		h.applyTVMetadata(media, details)
	}

	// Update in database
	if err := h.db.UpdateMedia(media); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update media"})
		return
	}

	c.JSON(http.StatusOK, media)
}

// POST /api/media/:id/metadata/refresh
// Force re-lookup metadata from TMDB using existing title/year
func (h *MetadataHandler) RefreshMetadata(c *gin.Context) {
	mediaID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	media, err := h.db.GetMediaByID(mediaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	// Search using existing title and year
	if media.Type == db.MediaTypeMovie {
		result, err := h.tmdb.SearchMovie(media.Title, media.Year)
		if err != nil || result == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No match found on TMDB"})
			return
		}

		// Get full details
		details, err := h.tmdb.GetMovieDetails(result.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch details"})
			return
		}

		h.applyMovieMetadata(media, details)
	} else if media.Type == db.MediaTypeTVShow {
		result, err := h.tmdb.SearchTV(media.Title, media.Year)
		if err != nil || result == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No match found on TMDB"})
			return
		}

		// Get full details
		details, err := h.tmdb.GetTVDetails(result.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch details"})
			return
		}

		h.applyTVMetadata(media, details)
	}

	// Update in database
	if err := h.db.UpdateMedia(media); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update media"})
		return
	}

	c.JSON(http.StatusOK, media)
}

// Helper functions
func (h *MetadataHandler) applyMovieMetadata(media *db.Media, details *tmdb.MovieDetails) {
	media.Title = details.Title
	media.OriginalTitle = details.OriginalTitle
	media.Overview = details.Overview
	media.Year = extractYear(details.ReleaseDate)
	media.Rating = details.VoteAverage
	media.Runtime = details.Runtime
	media.Genres = joinGenres(details.Genres)
	media.TMDbID = details.ID
	media.IMDbID = details.IMDbID
	media.PosterPath = details.PosterPath
	media.BackdropPath = details.BackdropPath
}

func (h *MetadataHandler) applyTVMetadata(media *db.Media, details *tmdb.TVDetails) {
	media.Title = details.Name
	media.OriginalTitle = details.OriginalName
	media.Overview = details.Overview
	media.Year = extractYear(details.FirstAirDate)
	media.Rating = details.VoteAverage
	media.Genres = joinGenres(details.Genres)
	media.TMDbID = details.ID
	media.PosterPath = details.PosterPath
	media.BackdropPath = details.BackdropPath

	// Extract IMDB ID if available
	if details.ExternalIDs != nil && details.ExternalIDs.IMDbID != "" {
		media.IMDbID = details.ExternalIDs.IMDbID
	}
}

func extractYear(date string) int {
	if len(date) >= 4 {
		year, _ := strconv.Atoi(date[:4])
		return year
	}
	return 0
}

func joinGenres(genres []tmdb.Genre) string {
	names := make([]string, len(genres))
	for i, g := range genres {
		names[i] = g.Name
	}
	return strings.Join(names, ", ")
}
