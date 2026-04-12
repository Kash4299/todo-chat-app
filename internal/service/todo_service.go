package service

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	todoRepo "github.com/Kash4299/todo-chat-app/internal/repository/todo"
)

type TodoService struct {
	repo todoRepo.ITodoRepository
}

func NewTodoService(repo todoRepo.ITodoRepository) ITodoService {
	return &TodoService{repo: repo}
}

func (s *TodoService) Create(todo *model.Todo) error {
	return s.repo.Create(todo)
}

func (s *TodoService) GetByID(id uint) (*model.Todo, error) {
	return s.repo.FindByID(id)
}

func (s *TodoService) GetByUserID(userID uint) ([]model.Todo, error) {
	return s.repo.FindByUserID(userID)
}

func (s *TodoService) Update(todo *model.Todo) error {
	return s.repo.Update(todo)
}

func (s *TodoService) Delete(id uint) error {
	return s.repo.Delete(id)
}
