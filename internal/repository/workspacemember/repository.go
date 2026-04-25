package workspacemember

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IWorkspaceMemberRepository interface {
	IsMember(workspaceID, userID uuid.UUID) (bool, error)
}

type WorkspaceMemberRepository struct {
	db *gorm.DB
}

func NewWorkspaceMemberRepository(db *gorm.DB) IWorkspaceMemberRepository {
	return &WorkspaceMemberRepository{db: db}
}

func (r *WorkspaceMemberRepository) IsMember(workspaceID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&model.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Count(&count).Error
	return count > 0, err
}
