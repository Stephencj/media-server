package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/db"
)

type ExtrasHandler struct {
	db *db.DB
}

func NewExtrasHandler(database *db.DB) *ExtrasHandler {
	return &ExtrasHandler{db: database}
}

// GetExtras returns all extras with pagination
func (h *ExtrasHandler) GetExtras(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	extras, total, err := h.db.GetAllExtras(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch extras"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  extras,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetExtra returns a single extra by ID
func (h *ExtrasHandler) GetExtra(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid extra ID"})
		return
	}

	extra, err := h.db.GetExtraByID(id)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extra not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch extra"})
		return
	}

	c.JSON(http.StatusOK, extra)
}

// GetExtraCategories returns all categories with counts
func (h *ExtrasHandler) GetExtraCategories(c *gin.Context) {
	categories, err := h.db.GetExtraCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// GetExtrasByCategory returns extras filtered by category
func (h *ExtrasHandler) GetExtrasByCategory(c *gin.Context) {
	category := db.ExtraCategory(c.Param("category"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	extras, total, err := h.db.GetExtrasByCategory(category, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch extras"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    extras,
		"total":    total,
		"category": category,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetMovieExtras returns extras for a specific movie
func (h *ExtrasHandler) GetMovieExtras(c *gin.Context) {
	movieID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid movie ID"})
		return
	}

	extras, err := h.db.GetExtrasByMovieID(movieID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch extras"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": extras})
}

// GetShowExtras returns extras for a specific TV show
func (h *ExtrasHandler) GetShowExtras(c *gin.Context) {
	showID, err := strconv.ParseInt(c.Param("showId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
		return
	}

	extras, err := h.db.GetExtrasByTVShowID(showID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch extras"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": extras})
}

// GetEpisodeExtras returns extras for a specific episode
func (h *ExtrasHandler) GetEpisodeExtras(c *gin.Context) {
	episodeID, err := strconv.ParseInt(c.Param("episodeId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid episode ID"})
		return
	}

	extras, err := h.db.GetExtrasByEpisodeID(episodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch extras"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": extras})
}

// GetRandomExtra returns a random extra, optionally filtered by category
func (h *ExtrasHandler) GetRandomExtra(c *gin.Context) {
	category := c.Query("category")

	extra, err := h.db.GetRandomExtra(category)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "No extras found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch random extra"})
		return
	}

	c.JSON(http.StatusOK, extra)
}
