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

func newTestProgressDB(t *testing.T) *db.DB {
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

func newProgressTestUser(t *testing.T, database *db.DB) int64 {
	t.Helper()
	user, err := database.CreateUser("tester", "tester@example.com", "x")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user.ID
}

// TestContinueWatchingReturnsEmptyArray verifies that /api/continue-watching
// returns {"items": []} instead of {"items": null} when there is no progress.
func TestContinueWatchingReturnsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := newTestProgressDB(t)
	userID := newProgressTestUser(t, database)

	handler := NewProgressHandler(database)

	r := gin.New()
	r.GET("/api/continue-watching", func(c *gin.Context) {
		c.Set("user_id", userID)
		handler.GetContinueWatching(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/continue-watching", nil)
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
