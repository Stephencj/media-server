package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/db"
)

type ChannelHandler struct {
	db *db.DB
}

func NewChannelHandler(database *db.DB) *ChannelHandler {
	return &ChannelHandler{db: database}
}

// CreateChannelRequest represents the request body for creating a channel
type CreateChannelRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// AddSourceRequest represents the request body for adding a source
type AddSourceRequest struct {
	SourceType  string                  `json:"source_type" binding:"required"`
	SourceID    *int64                  `json:"source_id"`
	SourceValue string                  `json:"source_value"`
	Weight      int                     `json:"weight"`
	Shuffle     *bool                   `json:"shuffle"` // Defaults to true if not provided
	Options     *db.ChannelSourceOptions `json:"options"` // Filtering options for shows
}

// ListChannels returns all channels for the current user
func (h *ChannelHandler) ListChannels(c *gin.Context) {
	userID := c.GetInt64("user_id")

	channels, err := h.db.GetUserChannels(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channels"})
		return
	}

	if channels == nil {
		channels = []db.Channel{}
	}

	c.JSON(http.StatusOK, gin.H{"items": channels})
}

// GetGuide returns a time-windowed schedule across all of the user's channels.
//
// Replaces the clients' N+1 fan-out (list channels, then one /now per channel)
// with a single request, and is the only endpoint that reports wall-clock air
// times rather than offsets into a cycle.
//
//	GET /api/channels/guide?start=<RFC3339>&hours=<1..12>
//
// Both params are optional: start defaults to the previous half-hour boundary
// so the grid lines up with its own column headings, hours defaults to 4.
func (h *ChannelHandler) GetGuide(c *gin.Context) {
	userID := c.GetInt64("user_id")

	start := time.Now()
	if raw := c.Query("start"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start must be an RFC3339 timestamp"})
			return
		}
		start = parsed
	}
	// Snap back to the half hour so column headings read 8:00, 8:30, 9:00.
	start = start.Truncate(30 * time.Minute)

	hours := 4
	if raw := c.Query("hours"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 12 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hours must be between 1 and 12"})
			return
		}
		hours = parsed
	}

	guide, err := h.db.GetChannelGuide(userID, start, start.Add(time.Duration(hours)*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build guide"})
		return
	}

	c.JSON(http.StatusOK, guide)
}

// CreateChannel creates a new channel
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	channel, err := h.db.CreateChannel(userID, req.Name, req.Description, req.Icon)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel"})
		return
	}

	c.JSON(http.StatusCreated, channel)
}

// GetChannel returns a specific channel
func (h *ChannelHandler) GetChannel(c *gin.Context) {
	userID := c.GetInt64("user_id")
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	channel, err := h.db.GetChannelByID(channelID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}

	// Verify ownership
	if channel.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, channel)
}

// UpdateChannel updates a channel's details
func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	userID := c.GetInt64("user_id")
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	// Verify ownership
	existing, err := h.db.GetChannelByID(channelID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}
	if existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	channel, err := h.db.UpdateChannel(channelID, req.Name, req.Description, req.Icon)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
		return
	}

	c.JSON(http.StatusOK, channel)
}

// DeleteChannel deletes a channel
func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	userID := c.GetInt64("user_id")
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	// Verify ownership
	existing, err := h.db.GetChannelByID(channelID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}
	if existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.db.DeleteChannel(channelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel deleted"})
}

// AddSource adds a content source to a channel
func (h *ChannelHandler) AddSource(c *gin.Context) {
	userID := c.GetInt64("user_id")
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	// Verify ownership
	existing, err := h.db.GetChannelByID(channelID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}
	if existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req AddSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate source type
	validTypes := map[string]bool{
		db.ChannelSourceSection:       true,
		db.ChannelSourcePlaylist:      true,
		db.ChannelSourceShow:          true,
		db.ChannelSourceMovie:         true,
		db.ChannelSourceExtraCategory: true,
	}
	if !validTypes[req.SourceType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source type"})
		return
	}

	// Default shuffle to true if not provided
	shuffle := true
	if req.Shuffle != nil {
		shuffle = *req.Shuffle
	}

	source, err := h.db.AddChannelSource(channelID, req.SourceType, req.SourceID, req.SourceValue, req.Weight, shuffle, req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add source"})
		return
	}

	c.JSON(http.StatusCreated, source)
}

// GetSources returns all sources for a channel
func (h *ChannelHandler) GetSources(c *gin.Context) {
	userID := c.GetInt64("user_id")
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	// Verify ownership
	existing, err := h.db.GetChannelByID(channelID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}
	if existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	sources, err := h.db.GetChannelSources(channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sources"})
		return
	}

	if sources == nil {
		sources = []db.ChannelSource{}
	}

	c.JSON(http.StatusOK, gin.H{"items": sources})
}

// DeleteSource removes a source from a channel
func (h *ChannelHandler) DeleteSource(c *gin.Context) {
	userID := c.GetInt64("user_id")
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}
	sourceID, err := strconv.ParseInt(c.Param("sourceId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source ID"})
		return
	}

	// Verify channel ownership
	existing, err := h.db.GetChannelByID(channelID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}
	if existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Verify source belongs to channel
	source, err := h.db.GetChannelSourceByID(sourceID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source not found"})
		return
	}
	if source.ChannelID != channelID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Source does not belong to this channel"})
		return
	}

	if err := h.db.DeleteChannelSource(sourceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete source"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Source removed"})
}

// RegenerateSchedule regenerates the channel's schedule
func (h *ChannelHandler) RegenerateSchedule(c *gin.Context) {
	userID := c.GetInt64("user_id")
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	// Verify ownership
	existing, err := h.db.GetChannelByID(channelID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}
	if existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.db.GenerateChannelSchedule(channelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate schedule: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule regenerated"})
}

// GetNowPlaying returns what's currently playing on a channel
func (h *ChannelHandler) GetNowPlaying(c *gin.Context) {
	userID := c.GetInt64("user_id")
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	// Verify ownership
	existing, err := h.db.GetChannelByID(channelID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}
	if existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	nowPlaying, err := h.db.GetChannelNowPlaying(channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get now playing"})
		return
	}

	// Add stream URL if something is playing
	if nowPlaying.NowPlaying != nil {
		streamType := string(nowPlaying.NowPlaying.MediaType)
		nowPlaying.StreamURL = "/api/stream/" + strconv.FormatInt(nowPlaying.NowPlaying.MediaID, 10) +
			"/direct?type=" + streamType + "&start=" + strconv.Itoa(nowPlaying.Elapsed)
	}

	c.JSON(http.StatusOK, nowPlaying)
}

// GetSchedule returns the full schedule for a channel
func (h *ChannelHandler) GetSchedule(c *gin.Context) {
	userID := c.GetInt64("user_id")
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	// Verify ownership
	existing, err := h.db.GetChannelByID(channelID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}
	if existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	items, total, err := h.db.GetChannelSchedule(channelID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schedule"})
		return
	}

	if items == nil {
		items = []db.ChannelScheduleItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetShowOptions returns available options for a TV show source
func (h *ChannelHandler) GetShowOptions(c *gin.Context) {
	userID := c.GetInt64("user_id")
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}
	showID, err := strconv.ParseInt(c.Param("showId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid show ID"})
		return
	}

	// Verify channel ownership
	existing, err := h.db.GetChannelByID(channelID)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}
	if existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	options, err := h.db.GetShowOptionsForChannel(showID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch show options"})
		return
	}

	c.JSON(http.StatusOK, options)
}
