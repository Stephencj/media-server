package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/db"
)

type PlaylistHandler struct {
	db *db.DB
}

func NewPlaylistHandler(database *db.DB) *PlaylistHandler {
	return &PlaylistHandler{db: database}
}

// CreatePlaylistRequest represents the request body for creating a playlist
type CreatePlaylistRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// ReorderRequest represents the request body for reordering playlist items
type ReorderRequest struct {
	ItemIDs []int64 `json:"item_ids" binding:"required"`
}

// GetPlaylists returns all playlists for the current user
func (h *PlaylistHandler) GetPlaylists(c *gin.Context) {
	userID := c.GetInt64("user_id")

	playlists, err := h.db.GetUserPlaylists(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlists"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": playlists})
}

// GetPlaylist returns a single playlist with its items
func (h *PlaylistHandler) GetPlaylist(c *gin.Context) {
	userID := c.GetInt64("user_id")
	playlistID, err := strconv.ParseInt(c.Param("playlistId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}

	playlist, err := h.db.GetPlaylistByID(playlistID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlist"})
		return
	}

	// Check ownership
	if playlist.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get playlist items
	items, err := h.db.GetPlaylistItems(playlistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlist items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"playlist": playlist,
		"items":    items,
	})
}

// CreatePlaylist creates a new playlist
func (h *PlaylistHandler) CreatePlaylist(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req CreatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	playlist, err := h.db.CreatePlaylist(userID, req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create playlist"})
		return
	}

	c.JSON(http.StatusCreated, playlist)
}

// UpdatePlaylist updates a playlist's name and description
func (h *PlaylistHandler) UpdatePlaylist(c *gin.Context) {
	userID := c.GetInt64("user_id")
	playlistID, err := strconv.ParseInt(c.Param("playlistId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}

	// Check ownership
	playlist, err := h.db.GetPlaylistByID(playlistID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlist"})
		return
	}
	if playlist.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req CreatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.db.UpdatePlaylist(playlistID, req.Name, req.Description); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Playlist updated"})
}

// DeletePlaylist deletes a playlist
func (h *PlaylistHandler) DeletePlaylist(c *gin.Context) {
	userID := c.GetInt64("user_id")
	playlistID, err := strconv.ParseInt(c.Param("playlistId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}

	// Check ownership
	playlist, err := h.db.GetPlaylistByID(playlistID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlist"})
		return
	}
	if playlist.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.db.DeletePlaylist(playlistID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Playlist deleted"})
}

// AddToPlaylist adds a media item to a playlist
func (h *PlaylistHandler) AddToPlaylist(c *gin.Context) {
	userID := c.GetInt64("user_id")
	playlistID, err := strconv.ParseInt(c.Param("playlistId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}
	mediaID, err := strconv.ParseInt(c.Param("mediaId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	// Check playlist ownership
	playlist, err := h.db.GetPlaylistByID(playlistID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlist"})
		return
	}
	if playlist.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get media type from query param, default to "movie"
	mediaType := db.MediaType(c.DefaultQuery("type", "movie"))

	if err := h.db.AddToPlaylist(playlistID, mediaID, mediaType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Added to playlist"})
}

// RemoveFromPlaylist removes a media item from a playlist
func (h *PlaylistHandler) RemoveFromPlaylist(c *gin.Context) {
	userID := c.GetInt64("user_id")
	playlistID, err := strconv.ParseInt(c.Param("playlistId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}
	mediaID, err := strconv.ParseInt(c.Param("mediaId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	// Check playlist ownership
	playlist, err := h.db.GetPlaylistByID(playlistID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlist"})
		return
	}
	if playlist.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get media type from query param, default to "movie"
	mediaType := db.MediaType(c.DefaultQuery("type", "movie"))

	if err := h.db.RemoveFromPlaylist(playlistID, mediaID, mediaType); err != nil {
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item not found in playlist"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove from playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Removed from playlist"})
}

// ReorderPlaylist reorders items in a playlist
func (h *PlaylistHandler) ReorderPlaylist(c *gin.Context) {
	userID := c.GetInt64("user_id")
	playlistID, err := strconv.ParseInt(c.Param("playlistId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}

	// Check playlist ownership
	playlist, err := h.db.GetPlaylistByID(playlistID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlist"})
		return
	}
	if playlist.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.db.ReorderPlaylistItems(playlistID, req.ItemIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Playlist reordered"})
}
