package services

import "todo/internal/repositories/todo"

type Hub struct {
	TodoRepository *todo.TodoRepository
}

func NewHub(todoRepository *todo.TodoRepository) *Hub {
	return &Hub{
		TodoRepository: todoRepository,
	}
}
