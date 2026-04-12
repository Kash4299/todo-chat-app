package service

import (
	"errors"

	"github.com/Kash4299/todo-chat-app/internal/model"
	userRepo "github.com/Kash4299/todo-chat-app/internal/repository/user"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo userRepo.IUserRepository
}

func NewUserService(repo userRepo.IUserRepository) IUserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(user *model.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	return s.repo.Create(user)
}

func (s *UserService) Login(email, password string) (*model.User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}
