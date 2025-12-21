package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/db"
)

type WatchlistHandler struct {
	db *db.DB
}

func NewWatchlistHandler(database *db.DB) *WatchlistHandler {
	return &WatchlistHandler{db: database}
}

type WatchlistRequest struct {
	MediaType string `json:"media_type" binding:"required,oneof=movie tvshow episode"`
}

// GetWatchlist returns the user's watchlist
func (h *WatchlistHandler) GetWatchlist(c *gin.Context) {
	userID, _ := c.Get("user_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	if limit > 100 {
		limit = 100
	}

	items, err := h.db.GetWatchlist(userID.(int64), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watchlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// AddToWatchlist adds a media item to the user's watchlist
func (h *WatchlistHandler) AddToWatchlist(c *gin.Context) {
	userID, _ := c.Get("user_id")
	mediaIDStr := c.Param("mediaId")

	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	var req WatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.db.AddToWatchlist(userID.(int64), mediaID, db.MediaType(req.MediaType))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to watchlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Added to watchlist"})
}

// RemoveFromWatchlist removes a media item from the user's watchlist
func (h *WatchlistHandler) RemoveFromWatchlist(c *gin.Context) {
	userID, _ := c.Get("user_id")
	mediaIDStr := c.Param("mediaId")
	mediaType := c.Query("type")

	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	if mediaType == "" {
		mediaType = "movie"
	}

	err = h.db.RemoveFromWatchlist(userID.(int64), mediaID, db.MediaType(mediaType))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove from watchlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Removed from watchlist"})
}

// CheckWatchlist checks if a media item is in the user's watchlist
func (h *WatchlistHandler) CheckWatchlist(c *gin.Context) {
	userID, _ := c.Get("user_id")
	mediaIDStr := c.Param("mediaId")
	mediaType := c.Query("type")

	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	if mediaType == "" {
		mediaType = "movie"
	}

	inWatchlist, err := h.db.IsInWatchlist(userID.(int64), mediaID, db.MediaType(mediaType))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check watchlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"in_watchlist": inWatchlist})
}

// MarkAsWatched marks a media item as watched (completed)
func (h *WatchlistHandler) MarkAsWatched(c *gin.Context) {
	userID, _ := c.Get("user_id")
	mediaIDStr := c.Param("mediaId")

	mediaID, err := strconv.ParseInt(mediaIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	var req WatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.db.MarkAsWatched(userID.(int64), mediaID, db.MediaType(req.MediaType))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as watched"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked as watched"})
}
