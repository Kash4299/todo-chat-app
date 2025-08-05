package services

import (
	"context"
	"fmt"
	"todo/common"
	"todo/internal/dtos/request"
	"todo/internal/model"
)

type UserService struct {
	Hub *Hub
}

func NewUserService(h *Hub) *UserService {
	return &UserService{
		Hub: h,
	}
}

func (s *UserService) CreateUser(ctx context.Context, request *request.CreateUserRequest) error {
	passwordHash, err := common.HashingArgon2(request.Password)
	if err != nil {
		return fmt.Errorf("hash password err : %v", err)
	}

	user := &model.User{
		Username: request.Username,
		Password: passwordHash,
		Email:    request.Email,
	}

	err = s.Hub.UserRepository.CreateUser(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) UserLogin(ctx context.Context, request *request.UserLoginRequest) error {
	// get user by username
	user, err := s.Hub.UserRepository.GetUserByUsername(ctx, request.Username)
	if err != nil {
		return err
	}

	compare, err := common.ComparePasswordAndHash(request.Password, user.Password)
	if err != nil {
		return err
	}

	if !compare {
		return fmt.Errorf("wrong password : %s", request.Password)
	}

	//TODO : jwt token

	return nil
}
