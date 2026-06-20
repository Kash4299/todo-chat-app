package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func corsTestRouter(origins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(origins))
	r.GET("/x", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func TestCORS_AllowedOrigin_EchoesOriginWithCredentials(t *testing.T) {
	r := corsTestRouter([]string{"https://app.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected echoed origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials=true for allowlisted origin, got %q", got)
	}
}

func TestCORS_DisallowedOrigin_NoAllowOriginHeader(t *testing.T) {
	r := corsTestRouter([]string{"https://app.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("disallowed origin must get no allow-origin header, got %q", got)
	}
}

func TestCORS_Preflight_AllowedOrigin_Returns204(t *testing.T) {
	r := corsTestRouter([]string{"https://app.example.com"})
	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("expected Access-Control-Allow-Methods on preflight")
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatal("expected echoed origin on preflight")
	}
}

func TestCORS_EmptyAllowlist_WildcardWithoutCredentials(t *testing.T) {
	r := corsTestRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://anything.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("empty allowlist (dev) should reply with *, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("must NOT send credentials alongside *, got %q", got)
	}
}
