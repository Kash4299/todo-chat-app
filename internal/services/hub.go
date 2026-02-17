package services

import (
	chatrepo "todo/internal/repositories/chat"
	todorepo "todo/internal/repositories/todo"
	userrepo "todo/internal/repositories/user"
)

type Hub struct {
	TodoRepository todorepo.ITodoRepository
	UserRepository userrepo.IUserRepository
	ChatRepository chatrepo.IChatRepository
}

func NewHub(todoRepository todorepo.ITodoRepository, userRepository userrepo.IUserRepository, chatRepository chatrepo.IChatRepository) *Hub {
	return &Hub{
		TodoRepository: todoRepository,
		UserRepository: userRepository,
		ChatRepository: chatRepository,
	}
}
