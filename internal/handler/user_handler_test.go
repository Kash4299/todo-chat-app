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

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Register(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserService) Login(email, password string) (*model.User, error) {
	args := m.Called(email, password)
	if args.Get(0) != nil {
		return args.Get(0).(*model.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestUserHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockUserService)
	handler := NewUserHandler(mockService)

	router := gin.Default()
	router.POST("/register", handler.Register)

	reqBody := RegisterRequest{
		Username: "testuser",
		Email:    "test@test.com",
		Password: "password123",
	}
	
	mockService.On("Register", mock.AnythingOfType("*model.User")).Return(nil)

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}
