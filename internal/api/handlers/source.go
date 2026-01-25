package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/db"
)

type SourceHandler struct {
	db *db.DB
}

func NewSourceHandler(database *db.DB) *SourceHandler {
	return &SourceHandler{db: database}
}

type CreateSourceRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Path     string `json:"path" binding:"required"`
	Type     string `json:"type" binding:"required,oneof=local smb nfs"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// GetSources returns all configured media sources
func (h *SourceHandler) GetSources(c *gin.Context) {
	sources, err := h.db.GetAllMediaSources()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sources"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sources": sources})
}

// CreateSource adds a new media source
func (h *SourceHandler) CreateSource(c *gin.Context) {
	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Clean and validate the path
	cleanPath := filepath.Clean(req.Path)

	// Security: Ensure path is under /media (the container's media mount point)
	if !strings.HasPrefix(cleanPath, "/media") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Path must be under /media. Use the file browser to select a valid path.",
		})
		return
	}

	// Prevent path traversal
	if strings.Contains(req.Path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	// Verify the path exists and is a directory
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Path does not exist: " + cleanPath,
				"hint":  "Use GET /api/files to browse available directories",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access path"})
		return
	}

	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path must be a directory"})
		return
	}

	source := &db.MediaSource{
		Name:     req.Name,
		Path:     cleanPath, // Use cleaned path
		Type:     req.Type,
		Username: req.Username,
		Password: req.Password,
		Enabled:  true,
	}

	created, err := h.db.CreateMediaSource(source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create source"})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// DeleteSource removes a media source
func (h *SourceHandler) DeleteSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source ID"})
		return
	}

	if err := h.db.DeleteMediaSource(id); err == db.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete source"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Source deleted"})
}
