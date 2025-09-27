package services

import (
	"context"
	"todo/internal/dtos/request"
	"todo/internal/model"
)

type TodoService struct {
	hub *Hub
}

func NewTodoService(h *Hub) *TodoService {
	return &TodoService{
		hub: h,
	}
}

func (s *TodoService) GetAllTodos(ctx context.Context) (*[]model.Todo, error) {
	todos, err := s.hub.TodoRepository.GetAllTodoList(ctx)
	if err != nil {
		return nil, err
	}

	return todos, nil
}

func (s *TodoService) CreateTodo(ctx context.Context, req *request.CreateTodoRequest) error {
	todo := &model.Todo{
		Title:       req.Title,
		Status:      req.Status,
		Priority:    req.Priority,
		Description: req.Description,
		Assigned:    req.Assigned,
	}

	return s.hub.TodoRepository.CreateTodo(ctx, todo)
}
