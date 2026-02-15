package user

import (
	"todo/internal/services"
)

type UserHandler struct {
	service services.IUserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{
		service: service,
	}
}
