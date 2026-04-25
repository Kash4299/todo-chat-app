package service_test

import (
	"errors"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockUserRepo struct {
	findByAuth0IDUser *model.User
	findByAuth0IDErr  error
	findByIDUser      *model.User
	findByIDErr       error
	createErr         error
	createCalls       int
	createdUser       *model.User
}

func (m *mockUserRepo) Create(user *model.User) error {
	m.createCalls++
	m.createdUser = user
	return m.createErr
}

func (m *mockUserRepo) FindByAuth0ID(auth0ID string) (*model.User, error) {
	if m.findByAuth0IDErr != nil {
		return nil, m.findByAuth0IDErr
	}
	return m.findByAuth0IDUser, nil
}

func (m *mockUserRepo) FindByID(id uuid.UUID) (*model.User, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.findByIDUser, nil
}

func TestUserService_SyncAuth0UserReturnsExistingUser(t *testing.T) {
	existingUser := &model.User{ID: uuid.New(), Auth0ID: "auth0|123"}
	repo := &mockUserRepo{findByAuth0IDUser: existingUser}

	svc := service.NewUserService(repo)
	got, err := svc.SyncAuth0User("auth0|123", "user@example.com", "User", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != existingUser {
		t.Fatal("expected existing user to be returned")
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected create not to be called, got %d", repo.createCalls)
	}
}

func TestUserService_SyncAuth0UserCreatesWhenNotFound(t *testing.T) {
	repo := &mockUserRepo{findByAuth0IDErr: gorm.ErrRecordNotFound}
	svc := service.NewUserService(repo)

	got, err := svc.SyncAuth0User("auth0|456", "new@example.com", "New User", "avatar")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected create to be called once, got %d", repo.createCalls)
	}
	if got != repo.createdUser {
		t.Fatal("expected created user to be returned")
	}
	if got.Auth0ID != "auth0|456" || got.Email != "new@example.com" || got.DisplayName != "New User" || got.AvatarURL != "avatar" {
		t.Fatalf("unexpected created user contents: %+v", got)
	}
}

func TestUserService_SyncAuth0UserReturnsLookupError(t *testing.T) {
	expectedErr := errors.New("database unavailable")
	repo := &mockUserRepo{findByAuth0IDErr: expectedErr}
	svc := service.NewUserService(repo)

	got, err := svc.SyncAuth0User("auth0|789", "user@example.com", "User", "")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected lookup error, got %v", err)
	}
	if got != nil {
		t.Fatal("expected nil user on lookup error")
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected create not to be called, got %d", repo.createCalls)
	}
}
