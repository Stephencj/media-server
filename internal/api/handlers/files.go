package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// FilesHandler handles file browsing operations
type FilesHandler struct {
	basePath string
}

// FileEntry represents a file or directory entry
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}

// NewFilesHandler creates a new files handler
// basePath is the root directory that can be browsed (e.g., "/media")
func NewFilesHandler(basePath string) *FilesHandler {
	return &FilesHandler{basePath: basePath}
}

// ListDirectory returns the contents of a directory
// GET /api/files?path=/media
func (h *FilesHandler) ListDirectory(c *gin.Context) {
	requestedPath := c.DefaultQuery("path", h.basePath)

	// Security: Clean and validate the path
	cleanPath := filepath.Clean(requestedPath)

	// Ensure the path is under the base path
	if !strings.HasPrefix(cleanPath, h.basePath) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Path must be under " + h.basePath,
		})
		return
	}

	// Prevent path traversal attacks
	if strings.Contains(requestedPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid path",
		})
		return
	}

	// Check if path exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Path not found: " + cleanPath,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to access path",
		})
		return
	}

	// Ensure it's a directory
	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Path is not a directory",
		})
		return
	}

	// Read directory contents
	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read directory",
		})
		return
	}

	// Build response
	files := make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		entryPath := filepath.Join(cleanPath, entry.Name())
		fileEntry := FileEntry{
			Name:  entry.Name(),
			Path:  entryPath,
			IsDir: entry.IsDir(),
		}

		// Get file size for regular files
		if !entry.IsDir() {
			if info, err := entry.Info(); err == nil {
				fileEntry.Size = info.Size()
			}
		}

		files = append(files, fileEntry)
	}

	c.JSON(http.StatusOK, gin.H{
		"path":    cleanPath,
		"entries": files,
		"parent":  filepath.Dir(cleanPath),
	})
}

// GetRoots returns the available root directories for browsing
// GET /api/files/roots
func (h *FilesHandler) GetRoots(c *gin.Context) {
	// Check what's available under the base path
	entries, err := os.ReadDir(h.basePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "Failed to read base path",
			"basePath": h.basePath,
		})
		return
	}

	roots := make([]FileEntry, 0)
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			roots = append(roots, FileEntry{
				Name:  entry.Name(),
				Path:  filepath.Join(h.basePath, entry.Name()),
				IsDir: true,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"basePath": h.basePath,
		"roots":    roots,
	})
}
