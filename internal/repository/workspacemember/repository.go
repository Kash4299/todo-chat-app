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
	ListWithUsers(workspaceID uuid.UUID, page, pageSize int) ([]model.WorkspaceMemberInfo, int64, error)
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

func (r *WorkspaceMemberRepository) ListWithUsers(workspaceID uuid.UUID, page, pageSize int) ([]model.WorkspaceMemberInfo, int64, error) {
	const selectCols = "users.id AS user_id, users.display_name, users.email, users.avatar_url, workspace_members.role, workspace_members.joined_at"
	const join = "JOIN users ON workspace_members.user_id = users.id"
	const where = "workspace_members.workspace_id = ?"

	var total int64
	if err := r.db.Model(&model.WorkspaceMember{}).Joins(join).Where(where, workspaceID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	var members []model.WorkspaceMemberInfo
	err := r.db.Model(&model.WorkspaceMember{}).
		Select(selectCols).
		Joins(join).
		Where(where, workspaceID).
		Order("workspace_members.joined_at ASC").
		Limit(pageSize).Offset(offset).
		Scan(&members).Error
	return members, total, err
}
