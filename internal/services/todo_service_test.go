package services

import (
	"context"
	"errors"
	"testing"

	"todo/internal/dtos/request"
	"todo/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ─── Mock Repository ─────────────────────────────────────────────
// Copy this pattern for any new repository mock in the future.

type MockTodoRepository struct {
	mock.Mock
}

func (m *MockTodoRepository) GetAllTodoList(ctx context.Context) (*[]model.Todo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]model.Todo), args.Error(1)
}

func (m *MockTodoRepository) CreateTodo(ctx context.Context, in *model.Todo) error {
	args := m.Called(ctx, in)
	return args.Error(0)
}

// ─── Helper ──────────────────────────────────────────────────────
// Use this pattern to create a TodoService with a mock repository.

func newTestTodoService(mockRepo *MockTodoRepository) *TodoService {
	hub := &Hub{TodoRepository: mockRepo}
	return NewTodoService(hub)
}

// ─── Tests ───────────────────────────────────────────────────────
// Follow this table-driven test pattern for every new feature.

func TestGetAllTodos(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(m *MockTodoRepository)
		wantErr   bool
		wantLen   int
	}{
		{
			name: "success - returns todos",
			mockSetup: func(m *MockTodoRepository) {
				todos := &[]model.Todo{
					{
						BaseModel:   model.BaseModel{ID: uuid.New()},
						Title:       "Test Todo 1",
						Status:      model.ToDo,
						Priority:    model.Normal,
						Description: "First test todo",
					},
					{
						BaseModel:   model.BaseModel{ID: uuid.New()},
						Title:       "Test Todo 2",
						Status:      model.InProgress,
						Priority:    model.High,
						Description: "Second test todo",
					},
				}
				m.On("GetAllTodoList", mock.Anything).Return(todos, nil)
			},
			wantErr: false,
			wantLen: 2,
		},
		{
			name: "success - returns empty list",
			mockSetup: func(m *MockTodoRepository) {
				todos := &[]model.Todo{}
				m.On("GetAllTodoList", mock.Anything).Return(todos, nil)
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "error - repository failure",
			mockSetup: func(m *MockTodoRepository) {
				m.On("GetAllTodoList", mock.Anything).Return(nil, errors.New("database connection failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTodoRepository)
			tt.mockSetup(mockRepo)

			svc := newTestTodoService(mockRepo)
			result, err := svc.GetAllTodos(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, *result, tt.wantLen)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCreateTodo(t *testing.T) {
	tests := []struct {
		name      string
		req       *request.CreateTodoRequest
		mockSetup func(m *MockTodoRepository)
		wantErr   bool
	}{
		{
			name: "success - creates todo",
			req: &request.CreateTodoRequest{
				Title:       "New Todo",
				Status:      model.ToDo,
				Priority:    model.Normal,
				Description: "A new todo item",
				Assigned:    uuid.New(),
			},
			mockSetup: func(m *MockTodoRepository) {
				m.On("CreateTodo", mock.Anything, mock.AnythingOfType("*model.Todo")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - repository failure",
			req: &request.CreateTodoRequest{
				Title:       "New Todo",
				Status:      model.ToDo,
				Priority:    model.Normal,
				Description: "A new todo item",
				Assigned:    uuid.New(),
			},
			mockSetup: func(m *MockTodoRepository) {
				m.On("CreateTodo", mock.Anything, mock.AnythingOfType("*model.Todo")).
					Return(errors.New("insert failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTodoRepository)
			tt.mockSetup(mockRepo)

			svc := newTestTodoService(mockRepo)
			err := svc.CreateTodo(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
