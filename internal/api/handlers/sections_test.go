package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/db"
)

func newTestSectionDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

// TestSectionIconAlwaysPresent verifies that section responses always include
// the "icon" key, even when empty, since iOS expects it.
func TestSectionIconAlwaysPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newTestSectionDB(t)
	handler := NewSectionHandler(database)

	// Create a section with an empty icon
	section := &db.Section{
		Name:        "Test",
		Slug:        "test",
		Icon:        "", // Empty icon
		SectionType: db.SectionTypeStandard,
		IsVisible:   true,
	}
	if err := database.CreateSection(section); err != nil {
		t.Fatalf("create section: %v", err)
	}

	r := gin.New()
	r.GET("/api/sections", handler.ListSections)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sections", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	// Parse the response and verify icon key exists
	var resp struct {
		Sections []map[string]interface{} `json:"sections"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Sections) == 0 {
		t.Fatal("no sections returned")
	}

	// Icon key must exist (even if empty string)
	if _, ok := resp.Sections[0]["icon"]; !ok {
		t.Error("icon key missing from section response")
	}
}

// TestListSectionsReturnsArray verifies that /api/sections returns
// {"sections": [...]} as an array, not {"sections": null}.
func TestListSectionsReturnsArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newTestSectionDB(t)
	handler := NewSectionHandler(database)

	r := gin.New()
	r.GET("/api/sections", handler.ListSections)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sections", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp struct {
		Sections json.RawMessage `json:"sections"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Sections must be an array (starts with [), not null
	if len(resp.Sections) == 0 || resp.Sections[0] != '[' {
		t.Errorf("sections = %s, want an array", string(resp.Sections))
	}
}

// TestSectionMediaReturnsEmptyArray verifies that section media endpoints
// return {"items": []} when no media exists, not {"items": null}.
func TestSectionMediaReturnsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newTestSectionDB(t)
	handler := NewSectionHandler(database)

	// Create a section with no media
	section := &db.Section{
		Name:        "Empty",
		Slug:        "empty",
		Icon:        "folder",
		SectionType: db.SectionTypeStandard,
		IsVisible:   true,
	}
	if err := database.CreateSection(section); err != nil {
		t.Fatalf("create section: %v", err)
	}

	r := gin.New()
	r.GET("/api/sections/slug/:slug/media", handler.GetSectionMediaBySlug)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sections/slug/empty/media", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp struct {
		Items json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Items must be an empty array [], not null
	if string(resp.Items) != "[]" {
		t.Errorf("items = %s, want []", string(resp.Items))
	}
}
