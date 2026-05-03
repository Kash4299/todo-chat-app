package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── mock ──────────────────────────────────────────────────────────────────────

type mockTaskService struct {
	createErr     error
	getErr        error
	listErr       error
	listTasks     []model.Task
	listTotal     int64
	updateTaskErr error
	updatedTask   *model.Task
	deleteTaskErr error
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

func (m *mockTaskService) ListByWorkspace(actorID, workspaceID uuid.UUID, page, pageSize int) ([]model.Task, int64, error) {
	return m.listTasks, m.listTotal, m.listErr
}

func (m *mockTaskService) UpdateTask(actorID, taskID uuid.UUID, input service.TaskUpdateInput) (*model.Task, error) {
	if m.updateTaskErr != nil {
		return nil, m.updateTaskErr
	}
	if m.updatedTask != nil {
		return m.updatedTask, nil
	}
	return &model.Task{ID: taskID}, nil
}

func (m *mockTaskService) DeleteTask(actorID, taskID uuid.UUID) error {
	return m.deleteTaskErr
}

// Compile-time: ensure mockTaskService satisfies ITaskService
var _ service.ITaskService = (*mockTaskService)(nil)

// ── Create ────────────────────────────────────────────────────────────────────

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

// ── GetByID ───────────────────────────────────────────────────────────────────

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

// ── List ──────────────────────────────────────────────────────────────────────

func TestTaskHandler_ListRejectsWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tasks?workspace_id="+uuid.NewString(), nil)

	h.List(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestTaskHandler_ListRejectsMissingWorkspaceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.List(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTaskHandler_ListMapsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{listErr: constants.ErrForbidden})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tasks?workspace_id="+uuid.NewString(), nil)
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.List(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestTaskHandler_ListSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{
		listTasks: []model.Task{{ID: uuid.New()}, {ID: uuid.New()}},
		listTotal: 2,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tasks?workspace_id="+uuid.NewString(), nil)
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ── UpdateTask ────────────────────────────────────────────────────────────────

func TestTaskHandler_UpdateTaskRejectsWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/"+uuid.NewString(), bytes.NewReader([]byte(`{"status":"DONE"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}

	h.UpdateTask(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestTaskHandler_UpdateTaskRejectsInvalidTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/bad", bytes.NewReader([]byte(`{"status":"DONE"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.UpdateTask(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTaskHandler_UpdateTaskMapsServiceErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{"success", nil, http.StatusOK},
		{"forbidden", constants.ErrForbidden, http.StatusForbidden},
		{"not found", constants.ErrTaskNotFound, http.StatusNotFound},
		{"invalid input", constants.ErrTaskInvalidInput, http.StatusBadRequest},
		{"internal", errors.New("unexpected"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewTaskHandler(&mockTaskService{updateTaskErr: tc.serviceErr})
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/"+uuid.NewString(), bytes.NewReader([]byte(`{"status":"DONE"}`)))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
			c.Set(middleware.UserIDContextKey, uuid.New())

			h.UpdateTask(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// ── DeleteTask ────────────────────────────────────────────────────────────────

func TestTaskHandler_DeleteTaskRejectsWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+uuid.NewString(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}

	h.DeleteTask(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestTaskHandler_DeleteTaskRejectsInvalidTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/bad", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.DeleteTask(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTaskHandler_DeleteTaskMapsServiceErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{"success", nil, http.StatusNoContent},
		{"forbidden", constants.ErrForbidden, http.StatusForbidden},
		{"not found", constants.ErrTaskNotFound, http.StatusNotFound},
		{"invalid input", constants.ErrTaskInvalidInput, http.StatusBadRequest},
		{"internal", errors.New("unexpected"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewTaskHandler(&mockTaskService{deleteTaskErr: tc.serviceErr})
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+uuid.NewString(), nil)
			c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
			c.Set(middleware.UserIDContextKey, uuid.New())

			h.DeleteTask(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}
