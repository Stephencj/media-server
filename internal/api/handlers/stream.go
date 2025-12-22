package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/pkg/ffmpeg"
)

type StreamHandler struct {
	db             *db.DB
	cfg            *config.Config
	sessionManager *ffmpeg.SessionManager
}

func NewStreamHandler(database *db.DB, cfg *config.Config) *StreamHandler {
	sm := ffmpeg.NewSessionManager(
		cfg.FFmpegPath,
		cfg.TranscodeDir,
		cfg.EnableHWAccel,
		cfg.HWAccelType,
	)

	return &StreamHandler{
		db:             database,
		cfg:            cfg,
		sessionManager: sm,
	}
}

// GetManifest returns the HLS manifest for a media item
func (h *StreamHandler) GetManifest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	mediaType := c.Query("type")
	var filePath string
	var duration int
	var resolution string

	switch mediaType {
	case "episode":
		episode, err := h.db.GetEpisodeByID(id)
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Episode not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch episode"})
			return
		}
		filePath = episode.FilePath
		duration = episode.Duration
		resolution = episode.Resolution
	case "extra":
		extra, err := h.db.GetExtraByID(id)
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Extra not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch extra"})
			return
		}
		filePath = extra.FilePath
		duration = extra.Duration
		resolution = extra.Resolution
	default:
		media, err := h.db.GetMediaByID(id)
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
			return
		}
		filePath = media.FilePath
		duration = media.Duration
		resolution = media.Resolution
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media file not found"})
		return
	}

	// Check if direct play is possible (H.264/HEVC in MP4/MKV)
	if h.canDirectPlay(filePath) {
		manifest := h.generateDirectPlayManifestForFile(filePath, duration, id, mediaType)
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
		c.String(http.StatusOK, manifest)
		return
	}

	// Need to transcode - check for existing manifest
	transcodeDir := filepath.Join(h.cfg.TranscodeDir, fmt.Sprintf("%d", id))
	manifestPath := filepath.Join(transcodeDir, "manifest.m3u8")

	// Check if transcode is complete
	if data, err := os.ReadFile(manifestPath); err == nil {
		if strings.Contains(string(data), "#EXT-X-ENDLIST") {
			// Transcode complete, serve the file
			c.Header("Content-Type", "application/vnd.apple.mpegurl")
			c.File(manifestPath)
			return
		}
	}

	// Start or get existing transcode session
	profile := ffmpeg.Profiles["1080p"]
	// Use resolution string to determine profile (e.g., "1920x1080")
	if resolution != "" && strings.Contains(resolution, "x") {
		parts := strings.Split(resolution, "x")
		if len(parts) == 2 {
			if height, err := strconv.Atoi(parts[1]); err == nil && height <= 720 {
				profile = ffmpeg.Profiles["720p"]
			}
		}
	}

	_, err = h.sessionManager.GetOrStartSession(id, filePath, profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transcoding: " + err.Error()})
		return
	}

	// Wait for initial segments (at least 2 for smooth playback)
	err = h.sessionManager.WaitForSegments(id, 2, 30*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transcoding timeout - " + err.Error()})
		return
	}

	// Serve the manifest (now has at least some segments)
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "no-cache")
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

	// Wait for segment if transcoding is in progress
	if h.sessionManager.IsTranscoding(id) {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(segmentPath); err == nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Segment not found"})
		return
	}

	c.Header("Content-Type", "video/MP2T")
	c.Header("Cache-Control", "max-age=86400")
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

	mediaType := c.Query("type")
	var filePath string

	switch mediaType {
	case "episode":
		// Look up episode from episodes table
		episode, err := h.db.GetEpisodeByID(id)
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Episode not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch episode"})
			return
		}
		filePath = episode.FilePath
	case "extra":
		// Look up from extras table
		extra, err := h.db.GetExtraByID(id)
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Extra not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch extra"})
			return
		}
		filePath = extra.FilePath
	default:
		// Look up from media table (movies)
		media, err := h.db.GetMediaByID(id)
		if err == db.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
			return
		}
		filePath = media.FilePath
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media file not found"})
		return
	}

	// Determine content type
	contentType := h.getContentType(filePath)
	c.Header("Content-Type", contentType)
	c.Header("Accept-Ranges", "bytes")

	c.File(filePath)
}

// StopTranscode stops an active transcode session
func (h *StreamHandler) StopTranscode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	h.sessionManager.StopSession(id)
	c.JSON(http.StatusOK, gin.H{"message": "Transcode stopped"})
}

// canDirectPlay checks if the file can be played directly on Apple TV
func (h *StreamHandler) canDirectPlay(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// MP4 and M4V are typically directly playable
	if ext == ".mp4" || ext == ".m4v" {
		return true
	}

	// For MKV, we'd need to probe the codec
	// For now, assume MKV needs transcoding (common case)
	return false
}

func (h *StreamHandler) generateDirectPlayManifest(media *db.Media, id int64) string {
	duration := media.Duration
	if duration == 0 {
		duration = 3600 // Default 1 hour
	}

	return fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:%d
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:VOD
#EXTINF:%d.0,
/api/stream/%d/direct
#EXT-X-ENDLIST
`, duration, duration, id)
}

func (h *StreamHandler) generateDirectPlayManifestForFile(filePath string, duration int, id int64, mediaType string) string {
	if duration == 0 {
		duration = 3600 // Default 1 hour
	}

	typeParam := ""
	if mediaType != "" {
		typeParam = "?type=" + mediaType
	}

	return fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:%d
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:VOD
#EXTINF:%d.0,
/api/stream/%d/direct%s
#EXT-X-ENDLIST
`, duration, duration, id, typeParam)
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
