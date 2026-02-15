package services

import (
	"context"
	"todo/internal/dtos/request"
	"todo/internal/model"
)

// ITodoService defines the contract for todo business logic
type ITodoService interface {
	GetAllTodos(ctx context.Context) (*[]model.Todo, error)
	CreateTodo(ctx context.Context, req *request.CreateTodoRequest) error
}

// IUserService defines the contract for user business logic
type IUserService interface {
	CreateUser(ctx context.Context, req *request.CreateUserRequest) error
	UserLogin(ctx context.Context, req *request.UserLoginRequest) (string, error)
}
