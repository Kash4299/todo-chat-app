package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockTaskService struct {
	createErr error
	getErr    error
}

func (m *mockTaskService) Create(actorID uuid.UUID, t *model.Task) error {
	return m.createErr
}

func (m *mockTaskService) GetByID(actorID, id uuid.UUID) (*model.Task, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return &model.Task{ID: id, WorkspaceID: uuid.New(), Title: "x"}, nil
}

func (m *mockTaskService) GetByWorkspace(workspaceID uuid.UUID) ([]model.Task, error) {
	return nil, nil
}

func (m *mockTaskService) Update(t *model.Task) error {
	return nil
}

func (m *mockTaskService) Delete(id uuid.UUID) error {
	return nil
}

func TestTaskHandler_CreateRejectsWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{})

	body := []byte(`{"workspace_id":"` + uuid.NewString() + `","title":"x"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestTaskHandler_GetByIDRejectsWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+uuid.NewString(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}

	h.GetByID(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestTaskHandler_GetByIDMapsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{getErr: constants.ErrForbidden})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+uuid.NewString(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.GetByID(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestTaskHandler_CreateRejectsInvalidWorkspaceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{})

	body := []byte(`{"workspace_id":"bad","title":"x"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTaskHandler_CreateMapsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{createErr: constants.ErrForbidden})

	body := []byte(`{"workspace_id":"` + uuid.NewString() + `","title":"x"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.Create(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestTaskHandler_GetByIDInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/not-a-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.GetByID(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
