package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockUserService struct {
	updateProfileErr  error
	updateProfileUser *model.User
}

func (m *mockUserService) SyncAuth0User(auth0ID, email, displayName, avatarURL string, emailVerified bool) (*model.User, error) {
	return &model.User{ID: uuid.New(), Email: email}, nil
}
func (m *mockUserService) GetByAuth0ID(auth0ID string) (*model.User, error) { return nil, nil }
func (m *mockUserService) GetByID(id uuid.UUID) (*model.User, error) {
	return &model.User{ID: id}, nil
}
func (m *mockUserService) UpdateProfile(userID uuid.UUID, displayName string, avatarURL, statusText *string) (*model.User, error) {
	if m.updateProfileErr != nil {
		return nil, m.updateProfileErr
	}
	if m.updateProfileUser != nil {
		return m.updateProfileUser, nil
	}
	return &model.User{ID: userID}, nil
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

func TestUserHandler_UpdateProfileRejectsWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewUserHandler(&mockUserService{}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", bytes.NewReader([]byte(`{"display_name":"test"}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateProfile(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUserHandler_UpdateProfileMapsServiceErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"display_name":"Alice"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "bad JSON",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid input",
			body:       `{"display_name":"ok"}`,
			serviceErr: constants.ErrUserInvalidInput,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "user not found domain error",
			body:       `{"display_name":"ok"}`,
			serviceErr: constants.ErrUserNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "user not found gorm error",
			body:       `{"display_name":"ok"}`,
			serviceErr: gorm.ErrRecordNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "internal error",
			body:       `{"display_name":"ok"}`,
			serviceErr: errors.New("unexpected db failure"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewUserHandler(&mockUserService{updateProfileErr: tc.serviceErr}, nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", bytes.NewReader([]byte(tc.body)))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set(middleware.UserIDContextKey, uuid.New())

			h.UpdateProfile(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

// Compile-time: ensure mockUserService satisfies IUserService
var _ service.IUserService = (*mockUserService)(nil)
