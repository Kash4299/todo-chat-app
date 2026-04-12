package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTodoService struct {
	mock.Mock
}

func (m *MockTodoService) Create(todo *model.Todo) error {
	args := m.Called(todo)
	if todo.ID == 0 {
		todo.ID = 1 // simulate creation
	}
	return args.Error(0)
}

func (m *MockTodoService) GetByID(id uint) (*model.Todo, error) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.Todo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTodoService) GetByUserID(userID uint) ([]model.Todo, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).([]model.Todo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTodoService) Update(todo *model.Todo) error {
	args := m.Called(todo)
	return args.Error(0)
}

func (m *MockTodoService) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestTodoHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockTodoService)
	handler := NewTodoHandler(mockService)

	router := gin.Default()
	router.POST("/todos", handler.Create)

	reqBody := CreateTodoRequest{
		UserID:      1,
		Title:       "Test Todo",
		Description: "Desc",
	}

	mockService.On("Create", mock.AnythingOfType("*model.Todo")).Return(nil)

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Test Todo", data["title"])
	mockService.AssertExpectations(t)
}
