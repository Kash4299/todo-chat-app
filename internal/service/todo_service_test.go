package service

import (
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTodoRepository
type MockTodoRepository struct {
	mock.Mock
}

func (m *MockTodoRepository) Create(todo *model.Todo) error {
	args := m.Called(todo)
	return args.Error(0)
}

func (m *MockTodoRepository) FindByID(id uint) (*model.Todo, error) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.Todo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTodoRepository) FindByUserID(userID uint) ([]model.Todo, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).([]model.Todo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTodoRepository) Update(todo *model.Todo) error {
	args := m.Called(todo)
	return args.Error(0)
}

func (m *MockTodoRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestTodoService_Create(t *testing.T) {
	mockRepo := new(MockTodoRepository)
	todoService := NewTodoService(mockRepo)

	todo := &model.Todo{
		UserID:      1,
		Title:       "Test Todo",
		Description: "Test Description",
	}

	mockRepo.On("Create", todo).Return(nil)

	err := todoService.Create(todo)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
