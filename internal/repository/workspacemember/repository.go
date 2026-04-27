package workspacemember

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IWorkspaceMemberRepository interface {
	WithTx(tx *gorm.DB) IWorkspaceMemberRepository
	IsMember(workspaceID, userID uuid.UUID) (bool, error)
	AddMember(workspaceID, userID uuid.UUID, role string) error
	GetRole(workspaceID, userID uuid.UUID) (string, error)
}

type WorkspaceMemberRepository struct {
	db *gorm.DB
}

func NewWorkspaceMemberRepository(db *gorm.DB) IWorkspaceMemberRepository {
	return &WorkspaceMemberRepository{db: db}
}

func (r *WorkspaceMemberRepository) WithTx(tx *gorm.DB) IWorkspaceMemberRepository {
	return &WorkspaceMemberRepository{db: tx}
}

func (r *WorkspaceMemberRepository) IsMember(workspaceID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&model.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *WorkspaceMemberRepository) AddMember(workspaceID, userID uuid.UUID, role string) error {
	return r.db.Create(&model.WorkspaceMember{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
	}).Error
}

func (r *WorkspaceMemberRepository) GetRole(workspaceID, userID uuid.UUID) (string, error) {
	var member model.WorkspaceMember
	err := r.db.
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		First(&member).Error
	if err != nil {
		return "", err
	}
	return member.Role, nil
}
