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

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(ctx context.Context, in *model.User) error {
	args := m.Called(ctx, in)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// ─── Helper ──────────────────────────────────────────────────────

func newTestUserService(mockRepo *MockUserRepository) *UserService {
	hub := &Hub{UserRepository: mockRepo}
	return NewUserService(hub)
}

// ─── Tests ───────────────────────────────────────────────────────

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name      string
		req       *request.CreateUserRequest
		mockSetup func(m *MockUserRepository)
		wantErr   bool
	}{
		{
			name: "success - creates user",
			req: &request.CreateUserRequest{
				Username: "testuser",
				Password: "securepassword123",
				Email:    "test@example.com",
			},
			mockSetup: func(m *MockUserRepository) {
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - repository failure",
			req: &request.CreateUserRequest{
				Username: "testuser",
				Password: "securepassword123",
				Email:    "test@example.com",
			},
			mockSetup: func(m *MockUserRepository) {
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("*model.User")).
					Return(errors.New("duplicate key"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			tt.mockSetup(mockRepo)

			svc := newTestUserService(mockRepo)
			err := svc.CreateUser(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserLogin(t *testing.T) {
	// Pre-hash a known password for the test
	// We need a valid argon2 hash to test the compare function
	userID := uuid.New()

	tests := []struct {
		name      string
		req       *request.UserLoginRequest
		mockSetup func(m *MockUserRepository)
		wantErr   bool
		wantToken bool
	}{
		{
			name: "error - user not found",
			req: &request.UserLoginRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			mockSetup: func(m *MockUserRepository) {
				m.On("GetUserByUsername", mock.Anything, "nonexistent").
					Return(nil, errors.New("user not found"))
			},
			wantErr:   true,
			wantToken: false,
		},
		{
			name: "error - wrong password",
			req: &request.UserLoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			mockSetup: func(m *MockUserRepository) {
				// Return a user with a hash that won't match "wrongpassword"
				m.On("GetUserByUsername", mock.Anything, "testuser").
					Return(&model.User{
						BaseModel: model.BaseModel{ID: userID},
						Username:  "testuser",
						Password:  "$argon2id$v=19$m=65536,t=3,p=2$invalidhash$invalidhash",
						Email:     "test@example.com",
					}, nil)
			},
			wantErr:   true,
			wantToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			tt.mockSetup(mockRepo)

			svc := newTestUserService(mockRepo)
			token, err := svc.UserLogin(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				if tt.wantToken {
					assert.NotEmpty(t, token)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
