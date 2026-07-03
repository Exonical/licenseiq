package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDocsHTMLDisablesDefaultFonts(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	MountDocs(r)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "withDefaultFonts: false") {
		t.Fatalf("expected docs html to disable default fonts")
	}
	if strings.Contains(body, "fonts.scalar.com") || strings.Contains(body, "cdn.jsdelivr.net") || strings.Contains(body, "unpkg.com") {
		t.Fatalf("expected no external font or cdn references in docs html")
	}
}
