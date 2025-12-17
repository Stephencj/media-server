package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
)

type StreamHandler struct {
	db  *db.DB
	cfg *config.Config
}

func NewStreamHandler(database *db.DB, cfg *config.Config) *StreamHandler {
	return &StreamHandler{db: database, cfg: cfg}
}

// GetManifest returns the HLS manifest for a media item
func (h *StreamHandler) GetManifest(c *gin.Context) {
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

	// Check if transcode exists or if direct play is possible
	transcodeDir := filepath.Join(h.cfg.TranscodeDir, fmt.Sprintf("%d", id))
	manifestPath := filepath.Join(transcodeDir, "manifest.m3u8")

	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		// TODO: Trigger transcoding if needed
		// For now, generate a simple manifest pointing to direct stream
		manifest := h.generateDirectPlayManifest(media, id)
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
		c.String(http.StatusOK, manifest)
		return
	}

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.File(manifestPath)
}

// GetSegment returns an HLS segment
func (h *StreamHandler) GetSegment(c *gin.Context) {
	idStr := c.Param("id")
	numStr := c.Param("num")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	_, err = h.db.GetMediaByID(id)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	transcodeDir := filepath.Join(h.cfg.TranscodeDir, fmt.Sprintf("%d", id))
	segmentPath := filepath.Join(transcodeDir, fmt.Sprintf("segment%s.ts", numStr))

	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Segment not found"})
		return
	}

	c.Header("Content-Type", "video/MP2T")
	c.File(segmentPath)
}

// GetSubtitle returns a subtitle file in VTT format
func (h *StreamHandler) GetSubtitle(c *gin.Context) {
	idStr := c.Param("id")
	lang := c.Param("lang")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	_, err = h.db.GetMediaByID(id)
	if err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	// Remove .vtt extension if present
	lang = strings.TrimSuffix(lang, ".vtt")

	transcodeDir := filepath.Join(h.cfg.TranscodeDir, fmt.Sprintf("%d", id))
	subtitlePath := filepath.Join(transcodeDir, fmt.Sprintf("subtitle_%s.vtt", lang))

	if _, err := os.Stat(subtitlePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subtitle not found"})
		return
	}

	c.Header("Content-Type", "text/vtt")
	c.File(subtitlePath)
}

// DirectPlay streams the original file directly
func (h *StreamHandler) DirectPlay(c *gin.Context) {
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

	// Check if file exists
	if _, err := os.Stat(media.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media file not found"})
		return
	}

	// Determine content type
	contentType := h.getContentType(media.FilePath)
	c.Header("Content-Type", contentType)
	c.Header("Accept-Ranges", "bytes")

	c.File(media.FilePath)
}

func (h *StreamHandler) generateDirectPlayManifest(media *db.Media, id int64) string {
	// Generate a simple HLS manifest for direct play
	duration := media.Duration
	if duration == 0 {
		duration = 3600 // Default 1 hour
	}

	return fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:%d
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:%d,
/api/stream/%d/direct
#EXT-X-ENDLIST
`, duration, duration, id)
}

func (h *StreamHandler) getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".mkv":
		return "video/x-matroska"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".m4v":
		return "video/x-m4v"
	default:
		return "application/octet-stream"
	}
}
