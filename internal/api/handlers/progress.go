package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/db"
)

type ProgressHandler struct {
	db *db.DB
}

func NewProgressHandler(database *db.DB) *ProgressHandler {
	return &ProgressHandler{db: database}
}

type UpdateProgressRequest struct {
	Position  int    `json:"position" binding:"required,min=0"`
	Duration  int    `json:"duration" binding:"required,min=0"`
	MediaType string `json:"media_type" binding:"required,oneof=movie tvshow episode"`
	Completed bool   `json:"completed"`
}

type ContinueWatchingItem struct {
	Media    *db.Media         `json:"media"`
	Progress *db.WatchProgress `json:"progress"`
}

// GetProgress returns the watch progress for a media item
func (h *ProgressHandler) GetProgress(c *gin.Context) {
	userID, _ := c.Get("user_id")
	mediaIDStr := c.Param("mediaId")
	mediaType := c.Query("type")

	if mediaType == "" {
		mediaType = "movie"
	}

	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	progress, err := h.db.GetWatchProgress(userID.(int64), mediaID, db.MediaType(mediaType))
	if err == db.ErrNotFound {
		// Return empty progress
		c.JSON(http.StatusOK, gin.H{
			"media_id":  mediaID,
			"position":  0,
			"duration":  0,
			"completed": false,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// UpdateProgress updates the watch progress for a media item
func (h *ProgressHandler) UpdateProgress(c *gin.Context) {
	userID, _ := c.Get("user_id")
	mediaIDStr := c.Param("mediaId")

	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	var req UpdateProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Auto-mark as completed if near the end (95%)
	completed := req.Completed
	if req.Duration > 0 && float64(req.Position)/float64(req.Duration) > 0.95 {
		completed = true
	}

	err = h.db.UpsertWatchProgress(
		userID.(int64),
		mediaID,
		db.MediaType(req.MediaType),
		req.Position,
		req.Duration,
		completed,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"media_id":  mediaID,
		"position":  req.Position,
		"duration":  req.Duration,
		"completed": completed,
	})
}

// GetContinueWatching returns in-progress media for the current user
func (h *ProgressHandler) GetContinueWatching(c *gin.Context) {
	userID, _ := c.Get("user_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if limit > 20 {
		limit = 20
	}

	progressItems, err := h.db.GetContinueWatching(userID.(int64), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch continue watching"})
		return
	}

	// Fetch media details for each progress item
	var items []ContinueWatchingItem
	for _, p := range progressItems {
		media, err := h.db.GetMediaByID(p.MediaID)
		if err != nil {
			continue
		}
		items = append(items, ContinueWatchingItem{
			Media:    media,
			Progress: p,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}
