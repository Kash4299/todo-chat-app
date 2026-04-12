package service

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/repository/user"
	"github.com/google/uuid"
)

type IUserService interface {
	SyncAuth0User(auth0ID, email, displayName, avatarURL string) (*model.User, error)
	GetByAuth0ID(auth0ID string) (*model.User, error)
	GetByID(id uuid.UUID) (*model.User, error)
}

type UserService struct {
	repo user.IUserRepository
}

func NewUserService(repo user.IUserRepository) IUserService {
	return &UserService{repo: repo}
}

func (s *UserService) SyncAuth0User(auth0ID, email, displayName, avatarURL string) (*model.User, error) {
	usr, err := s.repo.FindByAuth0ID(auth0ID)
	if err == nil {
		return usr, nil
	}

	newUser := &model.User{
		Auth0ID:     auth0ID,
		Email:       email,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}

	if err := s.repo.Create(newUser); err != nil {
		return nil, err
	}
	return newUser, nil
}

func (s *UserService) GetByAuth0ID(auth0ID string) (*model.User, error) {
	return s.repo.FindByAuth0ID(auth0ID)
}

func (s *UserService) GetByID(id uuid.UUID) (*model.User, error) {
	return s.repo.FindByID(id)
}
