package grpcserver

import (
	"context"
	"errors"
	"testing"

	"todo/internal/dtos/request"
	pb "todo/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ─── Mock Service ────────────────────────────────────────────────

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, req *request.CreateUserRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockUserService) UserLogin(ctx context.Context, req *request.UserLoginRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

// ─── Helper ──────────────────────────────────────────────────────

func newTestUserGRPCHandler(mockSvc *MockUserService) *UserGRPCHandler {
	return &UserGRPCHandler{
		userService: mockSvc,
	}
}

// ─── Tests ───────────────────────────────────────────────────────

func TestGRPC_CreateUser(t *testing.T) {
	tests := []struct {
		name      string
		req       *pb.CreateUserRequest
		mockSetup func(m *MockUserService)
		wantErr   bool
		wantCode  codes.Code
	}{
		{
			name: "success - creates user",
			req: &pb.CreateUserRequest{
				Username: "newuser",
				Password: "securepassword",
				Email:    "new@example.com",
			},
			mockSetup: func(m *MockUserService) {
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*request.CreateUserRequest")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - service failure",
			req: &pb.CreateUserRequest{
				Username: "existing",
				Password: "password",
				Email:    "exists@example.com",
			},
			mockSetup: func(m *MockUserService) {
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*request.CreateUserRequest")).
					Return(errors.New("duplicate key"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(MockUserService)
			tt.mockSetup(mockSvc)

			handler := newTestUserGRPCHandler(mockSvc)
			resp, err := handler.CreateUser(context.Background(), tt.req)

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

func TestGRPC_Login(t *testing.T) {
	tests := []struct {
		name      string
		req       *pb.LoginRequest
		mockSetup func(m *MockUserService)
		wantErr   bool
		wantCode  codes.Code
		wantToken string
	}{
		{
			name: "success - login returns token",
			req: &pb.LoginRequest{
				Username: "testuser",
				Password: "correctpassword",
			},
			mockSetup: func(m *MockUserService) {
				m.On("UserLogin", mock.Anything, mock.AnythingOfType("*request.UserLoginRequest")).
					Return("jwt-token-123", nil)
			},
			wantErr:   false,
			wantToken: "jwt-token-123",
		},
		{
			name: "error - invalid credentials",
			req: &pb.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			mockSetup: func(m *MockUserService) {
				m.On("UserLogin", mock.Anything, mock.AnythingOfType("*request.UserLoginRequest")).
					Return("", errors.New("wrong password"))
			},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
		{
			name: "error - user not found",
			req: &pb.LoginRequest{
				Username: "ghost",
				Password: "password",
			},
			mockSetup: func(m *MockUserService) {
				m.On("UserLogin", mock.Anything, mock.AnythingOfType("*request.UserLoginRequest")).
					Return("", errors.New("user not found"))
			},
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(MockUserService)
			tt.mockSetup(mockSvc)

			handler := newTestUserGRPCHandler(mockSvc)
			resp, err := handler.Login(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.wantToken, resp.Token)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
