package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/db"
)

type SectionHandler struct {
	db *db.DB
}

func NewSectionHandler(database *db.DB) *SectionHandler {
	return &SectionHandler{db: database}
}

// GET /api/sections
// List all sections (or only visible ones)
func (h *SectionHandler) ListSections(c *gin.Context) {
	visibleOnly := c.Query("visible") == "true"

	var sections []db.Section
	var err error

	if visibleOnly {
		sections, err = h.db.GetVisibleSections()
	} else {
		sections, err = h.db.GetAllSections()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sections"})
		return
	}

	// Optionally include media counts
	if c.Query("with_counts") == "true" {
		for i := range sections {
			_, count, _ := h.db.GetMediaBySectionID(sections[i].ID, 1, 0)
			sections[i].MediaCount = count
		}
	}

	c.JSON(http.StatusOK, gin.H{"sections": sections})
}

// GET /api/sections/:id
// Get section details
func (h *SectionHandler) GetSection(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section ID"})
		return
	}

	section, err := h.db.GetSectionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Section not found"})
		return
	}

	// Include rules if requested
	if c.Query("with_rules") == "true" {
		rules, _ := h.db.GetSectionRules(section.ID)
		section.Rules = rules
	}

	c.JSON(http.StatusOK, section)
}

// GET /api/sections/slug/:slug
// Get section by slug
func (h *SectionHandler) GetSectionBySlug(c *gin.Context) {
	slug := c.Param("slug")

	section, err := h.db.GetSectionBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Section not found"})
		return
	}

	// Include rules if requested
	if c.Query("with_rules") == "true" {
		rules, _ := h.db.GetSectionRules(section.ID)
		section.Rules = rules
	}

	c.JSON(http.StatusOK, section)
}

// POST /api/sections
// Create a new section
func (h *SectionHandler) CreateSection(c *gin.Context) {
	var section db.Section

	if err := c.ShouldBindJSON(&section); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if section.Name == "" || section.Slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and slug are required"})
		return
	}

	// Set defaults
	if section.SectionType == "" {
		section.SectionType = db.SectionTypeStandard
	}
	if section.Icon == "" {
		section.Icon = "folder"
	}
	section.IsVisible = true

	if err := h.db.CreateSection(&section); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create section"})
		return
	}

	c.JSON(http.StatusCreated, section)
}

// PUT /api/sections/:id
// Update a section
func (h *SectionHandler) UpdateSection(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section ID"})
		return
	}

	var section db.Section
	if err := c.ShouldBindJSON(&section); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	section.ID = id

	if err := h.db.UpdateSection(&section); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update section"})
		return
	}

	c.JSON(http.StatusOK, section)
}

// DELETE /api/sections/:id
// Delete a section
func (h *SectionHandler) DeleteSection(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section ID"})
		return
	}

	if err := h.db.DeleteSection(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete section"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Section deleted"})
}

// GET /api/sections/:id/media
// Get media in a section
func (h *SectionHandler) GetSectionMedia(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section ID"})
		return
	}

	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	media, total, err := h.db.GetMediaBySectionID(id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  media,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GET /api/sections/slug/:slug/media
// Get media by slug
func (h *SectionHandler) GetSectionMediaBySlug(c *gin.Context) {
	slug := c.Param("slug")

	section, err := h.db.GetSectionBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Section not found"})
		return
	}

	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	media, total, err := h.db.GetMediaBySectionID(section.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  media,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// POST /api/sections/:id/media
// Add media to section (manual assignment)
func (h *SectionHandler) AddMediaToSection(c *gin.Context) {
	sectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section ID"})
		return
	}

	var req struct {
		MediaID   int64        `json:"media_id" binding:"required"`
		MediaType db.MediaType `json:"media_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.AddMediaToSection(req.MediaID, req.MediaType, sectionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add media to section"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Media added to section"})
}

// DELETE /api/sections/:id/media/:mediaId
// Remove media from section
func (h *SectionHandler) RemoveMediaFromSection(c *gin.Context) {
	sectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section ID"})
		return
	}

	mediaID, err := strconv.ParseInt(c.Param("mediaId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	mediaType := db.MediaType(c.Query("type"))
	if mediaType == "" {
		mediaType = db.MediaTypeMovie
	}

	if err := h.db.RemoveMediaFromSection(mediaID, mediaType, sectionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove media from section"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Media removed from section"})
}

// GET /api/sections/:id/rules
// Get section rules
func (h *SectionHandler) GetSectionRules(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section ID"})
		return
	}

	rules, err := h.db.GetSectionRules(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// POST /api/sections/:id/rules
// Add a rule to a section
func (h *SectionHandler) AddSectionRule(c *gin.Context) {
	sectionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section ID"})
		return
	}

	var rule db.SectionRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.SectionID = sectionID

	if err := h.db.CreateSectionRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// DELETE /api/sections/:id/rules/:ruleId
// Delete a rule
func (h *SectionHandler) DeleteSectionRule(c *gin.Context) {
	ruleID, err := strconv.ParseInt(c.Param("ruleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule ID"})
		return
	}

	if err := h.db.DeleteSectionRule(ruleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted"})
}

// PUT /api/sections/reorder
// Reorder sections by updating their display_order values
func (h *SectionHandler) ReorderSections(c *gin.Context) {
	var req struct {
		SectionIDs []int64 `json:"section_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update display_order for each section based on array index
	for i, sectionID := range req.SectionIDs {
		section, err := h.db.GetSectionByID(sectionID)
		if err != nil {
			continue
		}

		section.DisplayOrder = i
		if err := h.db.UpdateSection(section); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update section order"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sections reordered successfully"})
}
