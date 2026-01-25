package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/db"
)

type SectionTemplateHandler struct {
	db *db.DB
}

func NewSectionTemplateHandler(database *db.DB) *SectionTemplateHandler {
	return &SectionTemplateHandler{db: database}
}

// SectionTemplate defines a template for creating sections
type SectionTemplate struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Icon        string             `json:"icon"`
	SectionType string             `json:"section_type"`
	Rules       []db.SectionRule   `json:"rules"`
	Variables   []TemplateVariable `json:"variables,omitempty"`
}

type TemplateVariable struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "string", "number", "year_range"
	Default     string `json:"default,omitempty"`
}

// GET /api/sections/templates
func (h *SectionTemplateHandler) GetTemplates(c *gin.Context) {
	templates := h.getTemplateList()
	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// POST /api/sections/from-template
func (h *SectionTemplateHandler) CreateFromTemplate(c *gin.Context) {
	var req struct {
		TemplateID string            `json:"template_id" binding:"required"`
		Name       string            `json:"name"`
		Slug       string            `json:"slug"`
		Variables  map[string]string `json:"variables"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get template
	templates := h.getTemplateList()
	var template *SectionTemplate
	for i, t := range templates {
		if t.ID == req.TemplateID {
			template = &templates[i]
			break
		}
	}

	if template == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// Create section from template
	section := &db.Section{
		Name:        req.Name,
		Slug:        req.Slug,
		Icon:        template.Icon,
		Description: template.Description,
		SectionType: template.SectionType,
		IsVisible:   true,
	}

	if section.Name == "" {
		section.Name = template.Name
	}
	if section.Slug == "" {
		section.Slug = generateSlug(section.Name)
	}

	// Create section
	if err := h.db.CreateSection(section); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create section"})
		return
	}

	// Create rules with variable substitution
	for _, rule := range template.Rules {
		newRule := db.SectionRule{
			SectionID: section.ID,
			Field:     rule.Field,
			Operator:  rule.Operator,
			Value:     rule.Value,
		}

		// Apply variable substitutions if provided
		if len(req.Variables) > 0 {
			newRule.Value = applyVariables(rule.Value, req.Variables)
		}

		if err := h.db.CreateSectionRule(&newRule); err != nil {
			continue
		}
	}

	c.JSON(http.StatusCreated, section)
}

// Helper functions
func (h *SectionTemplateHandler) getTemplateList() []SectionTemplate {
	return []SectionTemplate{
		{
			ID:          "4k-content",
			Name:        "4K / UHD Content",
			Description: "Movies and shows in 4K resolution",
			Icon:        "sparkles",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "resolution", Operator: db.OperatorContains, Value: "\"2160\""},
			},
		},
		{
			ID:          "highly-rated",
			Name:        "Highly Rated",
			Description: "Content with ratings above a threshold",
			Icon:        "star",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "rating", Operator: db.OperatorGreaterThan, Value: "8.0"},
			},
			Variables: []TemplateVariable{
				{Name: "min_rating", Description: "Minimum rating", Type: "number", Default: "8.0"},
			},
		},
		{
			ID:          "recent-releases",
			Name:        "Recent Releases",
			Description: "Movies from recent years",
			Icon:        "calendar",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "year", Operator: db.OperatorGreaterThan, Value: "2020"},
			},
			Variables: []TemplateVariable{
				{Name: "min_year", Description: "Minimum year", Type: "number", Default: "2020"},
			},
		},
		{
			ID:          "by-genre",
			Name:        "Genre Collection",
			Description: "Movies of a specific genre",
			Icon:        "film",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "type", Operator: db.OperatorEquals, Value: "\"movie\""},
				{Field: "genres", Operator: db.OperatorContains, Value: "\"Action\""},
			},
			Variables: []TemplateVariable{
				{Name: "genre", Description: "Genre name", Type: "string", Default: "Action"},
			},
		},
		{
			ID:          "by-decade",
			Name:        "By Decade",
			Description: "Movies from a specific decade",
			Icon:        "clock",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "type", Operator: db.OperatorEquals, Value: "\"movie\""},
				{Field: "year", Operator: db.OperatorInRange, Value: "[2010, 2019]"},
			},
			Variables: []TemplateVariable{
				{Name: "decade_start", Description: "Decade start year", Type: "number", Default: "2010"},
				{Name: "decade_end", Description: "Decade end year", Type: "number", Default: "2019"},
			},
		},
		{
			ID:          "hd-content",
			Name:        "HD Content",
			Description: "1080p content",
			Icon:        "tv",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "resolution", Operator: db.OperatorContains, Value: "\"1080\""},
			},
		},
		{
			ID:          "documentaries",
			Name:        "Documentaries",
			Description: "Documentary films",
			Icon:        "book",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "type", Operator: db.OperatorEquals, Value: "\"movie\""},
				{Field: "genres", Operator: db.OperatorContains, Value: "\"Documentary\""},
			},
		},
		{
			ID:          "classics",
			Name:        "Classic Films",
			Description: "Movies from before 1980",
			Icon:        "film",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "type", Operator: db.OperatorEquals, Value: "\"movie\""},
				{Field: "year", Operator: db.OperatorLessThan, Value: "1980"},
			},
			Variables: []TemplateVariable{
				{Name: "max_year", Description: "Maximum year", Type: "number", Default: "1980"},
			},
		},
		{
			ID:          "tv-shows",
			Name:        "TV Shows",
			Description: "All TV show content",
			Icon:        "tv",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "type", Operator: db.OperatorEquals, Value: "\"tvshow\""},
			},
		},
		{
			ID:          "animated",
			Name:        "Animated Content",
			Description: "Animation genre movies and shows",
			Icon:        "sparkles",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "genres", Operator: db.OperatorContains, Value: "\"Animation\""},
			},
		},
		{
			ID:          "family-friendly",
			Name:        "Family Friendly",
			Description: "Highly rated family content",
			Icon:        "star",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "genres", Operator: db.OperatorContains, Value: "\"Family\""},
				{Field: "rating", Operator: db.OperatorGreaterThan, Value: "7.0"},
			},
			Variables: []TemplateVariable{
				{Name: "min_rating", Description: "Minimum rating", Type: "number", Default: "7.0"},
			},
		},
		{
			ID:          "action-movies",
			Name:        "Action Movies",
			Description: "Action genre movies only",
			Icon:        "film",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "type", Operator: db.OperatorEquals, Value: "\"movie\""},
				{Field: "genres", Operator: db.OperatorContains, Value: "\"Action\""},
			},
		},
		{
			ID:          "comedy-movies",
			Name:        "Comedy Movies",
			Description: "Comedy genre movies only",
			Icon:        "film",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "type", Operator: db.OperatorEquals, Value: "\"movie\""},
				{Field: "genres", Operator: db.OperatorContains, Value: "\"Comedy\""},
			},
		},
		{
			ID:          "horror-movies",
			Name:        "Horror Movies",
			Description: "Horror genre movies only",
			Icon:        "film",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "type", Operator: db.OperatorEquals, Value: "\"movie\""},
				{Field: "genres", Operator: db.OperatorContains, Value: "\"Horror\""},
			},
		},
		{
			ID:          "sci-fi-movies",
			Name:        "Science Fiction",
			Description: "Sci-Fi genre movies only",
			Icon:        "film",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "type", Operator: db.OperatorEquals, Value: "\"movie\""},
				{Field: "genres", Operator: db.OperatorContains, Value: "\"Science Fiction\""},
			},
		},
		{
			ID:          "long-movies",
			Name:        "Epic Length Films",
			Description: "Movies over 2.5 hours long",
			Icon:        "clock",
			SectionType: db.SectionTypeSmart,
			Rules: []db.SectionRule{
				{Field: "type", Operator: db.OperatorEquals, Value: "\"movie\""},
				{Field: "runtime", Operator: db.OperatorGreaterThan, Value: "150"},
			},
			Variables: []TemplateVariable{
				{Name: "min_runtime", Description: "Minimum runtime (minutes)", Type: "number", Default: "150"},
			},
		},
	}
}

func generateSlug(name string) string {
	// Simple slug generation
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, "&", "and")
	// Remove any other special characters
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	// Remove consecutive dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	// Trim dashes from start and end
	slug = strings.Trim(slug, "-")
	return slug
}

func applyVariables(value string, variables map[string]string) string {
	// Simple variable substitution
	for k, v := range variables {
		placeholder := "{{" + k + "}}"
		value = strings.ReplaceAll(value, placeholder, v)
	}
	return value
}
