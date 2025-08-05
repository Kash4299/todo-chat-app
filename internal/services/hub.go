package services

import (
	"todo/internal/repositories/todo"
	"todo/internal/repositories/user"
)

type Hub struct {
	TodoRepository *todo.TodoRepository
	UserRepository *user.UserRepository
}

func NewHub(todoRepository *todo.TodoRepository, userRepository *user.UserRepository) *Hub {
	return &Hub{
		TodoRepository: todoRepository,
		UserRepository: userRepository,
	}
}
