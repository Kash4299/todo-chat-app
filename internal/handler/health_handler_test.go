package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/handler"
	"github.com/gin-gonic/gin"
)

type mockHealthService struct{ readyErr error }

func (m mockHealthService) Ready(ctx context.Context) error { return m.readyErr }

func TestHealthHandler_Healthz_Always200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Even when the dependency is down, liveness must report OK.
	h := handler.NewHealthHandler(mockHealthService{readyErr: errors.New("db down")})
	r := gin.New()
	r.GET("/healthz", h.Healthz)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("healthz must be 200 regardless of dependencies, got %d", w.Code)
	}
}

func TestHealthHandler_Readyz_200WhenReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.NewHealthHandler(mockHealthService{readyErr: nil})
	r := gin.New()
	r.GET("/readyz", h.Readyz)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when ready, got %d", w.Code)
	}
}

func TestHealthHandler_Readyz_503WhenDBDown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.NewHealthHandler(mockHealthService{readyErr: errors.New("db down")})
	r := gin.New()
	r.GET("/readyz", h.Readyz)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when DB is down, got %d", w.Code)
	}
}
