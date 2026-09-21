package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// /channels/guide sits alongside /channels/:id. Older routers panic on a
// static segment colliding with a param at the same level; this pins the
// behaviour so the route ordering can't silently regress.
func TestGuideRouteDoesNotCollideWithIDRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	g := r.Group("/api/channels")
	g.GET("", func(c *gin.Context) { c.String(200, "list") })
	g.GET("/guide", func(c *gin.Context) { c.String(200, "guide") })
	g.GET("/:id", func(c *gin.Context) { c.String(200, "id:"+c.Param("id")) })
	g.GET("/:id/now", func(c *gin.Context) { c.String(200, "now:"+c.Param("id")) })

	for _, tc := range []struct{ path, want string }{
		{"/api/channels", "list"},
		{"/api/channels/guide", "guide"},
		{"/api/channels/42", "id:42"},
		{"/api/channels/42/now", "now:42"},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != http.StatusOK || w.Body.String() != tc.want {
			t.Errorf("%s -> %d %q, want 200 %q", tc.path, w.Code, w.Body.String(), tc.want)
		}
	}
}
