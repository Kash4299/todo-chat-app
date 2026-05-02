package service

import (
	"errors"
	"strings"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/repository/task"
	"github.com/Kash4299/todo-chat-app/internal/repository/workspacemember"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ITaskService interface {
	Create(actorID uuid.UUID, t *model.Task) error
	GetByID(actorID, id uuid.UUID) (*model.Task, error)
	GetByWorkspace(workspaceID uuid.UUID) ([]model.Task, error)
	Update(t *model.Task) error
	Delete(id uuid.UUID) error
}

type TaskService struct {
	repo                task.ITaskRepository
	workspaceMemberRepo workspacemember.IWorkspaceMemberRepository
}

func NewTaskService(
	repo task.ITaskRepository,
	workspaceMemberRepo workspacemember.IWorkspaceMemberRepository,
) ITaskService {
	return &TaskService{
		repo:                repo,
		workspaceMemberRepo: workspaceMemberRepo,
	}
}

func (s *TaskService) Create(actorID uuid.UUID, t *model.Task) error {
	if actorID == uuid.Nil || t == nil || t.WorkspaceID == uuid.Nil {
		return constants.ErrTaskInvalidInput
	}

	ok, err := s.workspaceMemberRepo.IsMember(t.WorkspaceID, actorID)
	if err != nil {
		return err
	}
	if !ok {
		return constants.ErrForbidden
	}

	t.CreatedBy = actorID
	t.Status = "TODO"
	t.Priority = strings.ToUpper(strings.TrimSpace(t.Priority))
	if t.Priority == "" {
		t.Priority = "MEDIUM"
	}
	switch t.Priority {
	case "LOW", "MEDIUM", "HIGH", "URGENT":
	default:
		return constants.ErrTaskInvalidInput
	}
	return s.repo.Create(t)
}

func (s *TaskService) GetByID(actorID, id uuid.UUID) (*model.Task, error) {
	if actorID == uuid.Nil || id == uuid.Nil {
		return nil, constants.ErrTaskInvalidInput
	}
	task, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrTaskNotFound
		}
		return nil, err
	}

	ok, err := s.workspaceMemberRepo.IsMember(task.WorkspaceID, actorID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, constants.ErrForbidden
	}
	return task, nil
}

func (s *TaskService) GetByWorkspace(workspaceID uuid.UUID) ([]model.Task, error) {
	return s.repo.FindByWorkspace(workspaceID)
}

func (s *TaskService) Update(t *model.Task) error {
	return s.repo.Update(t)
}

func (s *TaskService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
