package service

import (
	"github.com/Kash4299/todo-chat-app/internal/constants"
	workspaceMemberRepo "github.com/Kash4299/todo-chat-app/internal/repository/workspacemember"
	"github.com/google/uuid"
)

type IWorkspaceMemberService interface {
	GetRole(workspaceID, userID uuid.UUID) (string, error)
}

type WorkspaceMemberService struct {
	workspaceMemberRepo workspaceMemberRepo.IWorkspaceMemberRepository
}

func NewWorkspaceMemberService(workspaceMemberRepo workspaceMemberRepo.IWorkspaceMemberRepository) IWorkspaceMemberService {
	return &WorkspaceMemberService{
		workspaceMemberRepo: workspaceMemberRepo,
	}
}

func (s *WorkspaceMemberService) GetRole(workspaceID, userID uuid.UUID) (string, error) {
	if workspaceID == uuid.Nil || userID == uuid.Nil {
		return "", constants.ErrForbidden
	}
	return s.workspaceMemberRepo.GetRole(workspaceID, userID)
}
