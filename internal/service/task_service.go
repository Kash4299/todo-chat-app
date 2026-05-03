package service

import (
	"errors"
	"strings"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/repository/task"
	"github.com/Kash4299/todo-chat-app/internal/repository/workspacemember"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskUpdateInput carries the fields a caller wants to change. nil = skip, non-nil = apply.
// For DueDate and AssigneeID: empty string clears the field, non-empty sets it.
type TaskUpdateInput struct {
	Title       *string
	Description *string
	Status      *string
	Priority    *string
	DueDate     *string // nil=skip, ""=clear, "RFC3339"=set
	AssigneeID  *string // nil=skip, ""=unassign, "<uuid>"=assign
}

type ITaskService interface {
	Create(actorID uuid.UUID, t *model.Task) error
	GetByID(actorID, id uuid.UUID) (*model.Task, error)
	ListByWorkspace(actorID, workspaceID uuid.UUID, page, pageSize int) ([]model.Task, int64, error)
	UpdateTask(actorID, taskID uuid.UUID, input TaskUpdateInput) (*model.Task, error)
	DeleteTask(actorID, taskID uuid.UUID) error
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

func (s *TaskService) ListByWorkspace(actorID, workspaceID uuid.UUID, page, pageSize int) ([]model.Task, int64, error) {
	if actorID == uuid.Nil || workspaceID == uuid.Nil {
		return nil, 0, constants.ErrTaskInvalidInput
	}

	ok, err := s.workspaceMemberRepo.IsMember(workspaceID, actorID)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, constants.ErrForbidden
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	total, err := s.repo.CountByWorkspace(workspaceID)
	if err != nil {
		return nil, 0, err
	}

	tasks, err := s.repo.FindByWorkspace(workspaceID, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func (s *TaskService) UpdateTask(actorID, taskID uuid.UUID, input TaskUpdateInput) (*model.Task, error) {
	if actorID == uuid.Nil || taskID == uuid.Nil {
		return nil, constants.ErrTaskInvalidInput
	}

	t, err := s.repo.FindByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrTaskNotFound
		}
		return nil, err
	}

	ok, err := s.workspaceMemberRepo.IsMember(t.WorkspaceID, actorID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, constants.ErrForbidden
	}

	changed := false

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return nil, constants.ErrTaskInvalidInput
		}
		if t.Title != title {
			t.Title = title
			changed = true
		}
	}

	if input.Description != nil {
		if t.Description != *input.Description {
			t.Description = *input.Description
			changed = true
		}
	}

	if input.Status != nil {
		status := strings.ToUpper(strings.TrimSpace(*input.Status))
		switch status {
		case "TODO", "IN_PROGRESS", "DONE":
		default:
			return nil, constants.ErrTaskInvalidInput
		}
		if t.Status != status {
			t.Status = status
			changed = true
		}
	}

	if input.Priority != nil {
		priority := strings.ToUpper(strings.TrimSpace(*input.Priority))
		switch priority {
		case "LOW", "MEDIUM", "HIGH", "URGENT":
		default:
			return nil, constants.ErrTaskInvalidInput
		}
		if t.Priority != priority {
			t.Priority = priority
			changed = true
		}
	}

	if input.DueDate != nil {
		if *input.DueDate == "" {
			t.DueDate = nil
			changed = true
		} else {
			parsed, err := time.Parse(time.RFC3339, *input.DueDate)
			if err != nil {
				return nil, constants.ErrTaskInvalidInput
			}
			t.DueDate = &parsed
			changed = true
		}
	}

	if input.AssigneeID != nil {
		if *input.AssigneeID == "" {
			t.AssigneeID = nil
			changed = true
		} else {
			id, err := uuid.Parse(*input.AssigneeID)
			if err != nil || id == uuid.Nil {
				return nil, constants.ErrTaskInvalidInput
			}
			t.AssigneeID = &id
			changed = true
		}
	}

	if !changed {
		return t, nil
	}

	if err := s.repo.Update(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TaskService) DeleteTask(actorID, taskID uuid.UUID) error {
	if actorID == uuid.Nil || taskID == uuid.Nil {
		return constants.ErrTaskInvalidInput
	}

	t, err := s.repo.FindByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrTaskNotFound
		}
		return err
	}

	ok, err := s.workspaceMemberRepo.IsMember(t.WorkspaceID, actorID)
	if err != nil {
		return err
	}
	if !ok {
		return constants.ErrForbidden
	}

	return s.repo.Delete(taskID)
}
