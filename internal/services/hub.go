package services

import (
	todorepo "todo/internal/repositories/todo"
	userrepo "todo/internal/repositories/user"
)

type Hub struct {
	TodoRepository todorepo.ITodoRepository
	UserRepository userrepo.IUserRepository
}

func NewHub(todoRepository todorepo.ITodoRepository, userRepository userrepo.IUserRepository) *Hub {
	return &Hub{
		TodoRepository: todoRepository,
		UserRepository: userRepository,
	}
}
