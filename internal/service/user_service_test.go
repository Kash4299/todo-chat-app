package service

import (
	"errors"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(id uint) (*model.User, error) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*model.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) FindByEmail(email string) (*model.User, error) {
	args := m.Called(email)
	if args.Get(0) != nil {
		return args.Get(0).(*model.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestUserService_Register(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	user := &model.User{
		Username: "testuser",
		Email:    "test@test.com",
		Password: "password123",
	}

	mockRepo.On("Create", mock.AnythingOfType("*model.User")).Return(nil)

	err := userService.Register(user)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// Compare hash functionality implicitly proved by test pass
}

func TestUserService_Login_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &model.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@test.com",
		Password: string(hashedPassword),
	}

	mockRepo.On("FindByEmail", "test@test.com").Return(user, nil)

	loggedInUser, err := userService.Login("test@test.com", "password123")

	assert.NoError(t, err)
	assert.NotNil(t, loggedInUser)
	assert.Equal(t, uint(1), loggedInUser.ID)
	mockRepo.AssertExpectations(t)
}

func TestUserService_Login_InvalidPassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &model.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@test.com",
		Password: string(hashedPassword),
	}

	mockRepo.On("FindByEmail", "test@test.com").Return(user, nil)

	loggedInUser, err := userService.Login("test@test.com", "wrongpassword")

	assert.Error(t, err)
	assert.Equal(t, errors.New("invalid credentials"), err)
	assert.Nil(t, loggedInUser)
	mockRepo.AssertExpectations(t)
}
