package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockUserService struct{}

func (m *mockUserService) SyncAuth0User(auth0ID, email, displayName, avatarURL string, emailVerified bool) (*model.User, error) {
	return &model.User{ID: uuid.New(), Email: email}, nil
}
func (m *mockUserService) GetByAuth0ID(auth0ID string) (*model.User, error) { return nil, nil }
func (m *mockUserService) GetByID(id uuid.UUID) (*model.User, error) {
	return &model.User{ID: id}, nil
}

func TestUserHandler_GetMeRejectsWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewUserHandler(&mockUserService{}, &config.Config{WebhookSecret: "s"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)

	h.GetMe(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUserHandler_GetMeReturnsUserFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewUserHandler(&mockUserService{}, &config.Config{WebhookSecret: "s"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.GetMe(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUserHandler_SyncAuth0UserRequiresConfiguredSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewUserHandler(&mockUserService{}, &config.Config{WebhookSecret: ""})

	w := httptest.NewRecorder()
	body := []byte(`{"auth0_id":"auth0|x","email":"a@example.com","display_name":"A"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/auth0/users.sync", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.SyncAuth0User(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestUserHandler_SyncAuth0UserRejectsInvalidSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewUserHandler(&mockUserService{}, &config.Config{WebhookSecret: "expected"})

	w := httptest.NewRecorder()
	body := []byte(`{"auth0_id":"auth0|x","email":"a@example.com","display_name":"A"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/auth0/users.sync", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Webhook-Secret", "wrong")

	h.SyncAuth0User(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUserHandler_SyncAuth0UserSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewUserHandler(&mockUserService{}, &config.Config{WebhookSecret: "secret"})

	w := httptest.NewRecorder()
	body := []byte(`{"auth0_id":"auth0|x","email":"a@example.com","display_name":"A"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/auth0/users.sync", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Webhook-Secret", "secret")

	h.SyncAuth0User(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Compile-time: ensure mockUserService satisfies IUserService
var _ service.IUserService = (*mockUserService)(nil)
