package service

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/repository/task"
	"github.com/google/uuid"
)

type ITaskService interface {
	Create(task *model.Task) error
	GetByID(id uuid.UUID) (*model.Task, error)
	GetByUserID(userID uuid.UUID) ([]model.Task, error)
	Update(task *model.Task) error
	Delete(id uuid.UUID) error
}

type TaskService struct {
	repo task.ITaskRepository
}

func NewTaskService(repo task.ITaskRepository) ITaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) Create(t *model.Task) error {
	return s.repo.Create(t)
}

func (s *TaskService) GetByID(id uuid.UUID) (*model.Task, error) {
	return s.repo.FindByID(id)
}

func (s *TaskService) GetByUserID(userID uuid.UUID) ([]model.Task, error) {
	return s.repo.FindByCreatedBy(userID)
}

func (s *TaskService) Update(t *model.Task) error {
	return s.repo.Update(t)
}

func (s *TaskService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
