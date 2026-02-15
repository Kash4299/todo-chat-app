package grpcserver

import (
	"context"
	"errors"
	"testing"

	"todo/internal/dtos/request"
	"todo/internal/model"
	pb "todo/proto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ─── Mock Service ────────────────────────────────────────────────
// Copy this pattern for any new service mock in the future.

type MockTodoService struct {
	mock.Mock
}

func (m *MockTodoService) GetAllTodos(ctx context.Context) (*[]model.Todo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]model.Todo), args.Error(1)
}

func (m *MockTodoService) CreateTodo(ctx context.Context, req *request.CreateTodoRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// ─── Helper ──────────────────────────────────────────────────────

func newTestTodoGRPCHandler(mockSvc *MockTodoService) *TodoGRPCHandler {
	return &TodoGRPCHandler{
		todoService: mockSvc,
	}
}

// ─── Tests ───────────────────────────────────────────────────────

func TestGRPC_GetAllTodos(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(m *MockTodoService)
		wantErr   bool
		wantCode  codes.Code
		wantLen   int
	}{
		{
			name: "success - returns todos",
			mockSetup: func(m *MockTodoService) {
				todos := &[]model.Todo{
					{
						BaseModel:   model.BaseModel{ID: uuid.New()},
						Title:       "gRPC Todo",
						Status:      model.ToDo,
						Priority:    model.High,
						Description: "Test via gRPC",
					},
				}
				m.On("GetAllTodos", mock.Anything).Return(todos, nil)
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "success - empty list",
			mockSetup: func(m *MockTodoService) {
				todos := &[]model.Todo{}
				m.On("GetAllTodos", mock.Anything).Return(todos, nil)
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "error - service failure",
			mockSetup: func(m *MockTodoService) {
				m.On("GetAllTodos", mock.Anything).Return(nil, errors.New("db down"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(MockTodoService)
			tt.mockSetup(mockSvc)

			handler := newTestTodoGRPCHandler(mockSvc)
			resp, err := handler.GetAllTodos(context.Background(), &pb.GetAllTodosRequest{})

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.Todos, tt.wantLen)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestGRPC_CreateTodo(t *testing.T) {
	validUUID := uuid.New().String()

	tests := []struct {
		name      string
		req       *pb.CreateTodoRequest
		mockSetup func(m *MockTodoService)
		wantErr   bool
		wantCode  codes.Code
	}{
		{
			name: "success - creates todo",
			req: &pb.CreateTodoRequest{
				Title:       "New Todo",
				Status:      "todo",
				Priority:    1,
				Description: "A test todo",
				Assigned:    validUUID,
			},
			mockSetup: func(m *MockTodoService) {
				m.On("CreateTodo", mock.Anything, mock.AnythingOfType("*request.CreateTodoRequest")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - invalid UUID",
			req: &pb.CreateTodoRequest{
				Title:       "Bad Todo",
				Status:      "todo",
				Priority:    0,
				Description: "Invalid assigned",
				Assigned:    "not-a-uuid",
			},
			mockSetup: func(m *MockTodoService) {
				// No mock setup — should fail before calling service
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "error - service failure",
			req: &pb.CreateTodoRequest{
				Title:       "Failing Todo",
				Status:      "todo",
				Priority:    0,
				Description: "Will fail",
				Assigned:    validUUID,
			},
			mockSetup: func(m *MockTodoService) {
				m.On("CreateTodo", mock.Anything, mock.AnythingOfType("*request.CreateTodoRequest")).
					Return(errors.New("insert failed"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(MockTodoService)
			tt.mockSetup(mockSvc)

			handler := newTestTodoGRPCHandler(mockSvc)
			resp, err := handler.CreateTodo(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Contains(t, resp.Message, "successfully")
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
