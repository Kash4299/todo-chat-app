package workspace

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IWorkspaceRepository interface {
	WithTx(tx *gorm.DB) IWorkspaceRepository
	Create(ws *model.Workspace) error
	FindByID(id uuid.UUID) (*model.Workspace, error)
	// FindByUserID returns a page of workspaces the user belongs to, sorted by created_at DESC.
	// page is 1-indexed. pageSize is the number of items per page.
	FindByUserID(userID uuid.UUID, page, pageSize int) ([]model.Workspace, int64, error)
	Delete(id uuid.UUID) error
}

type WorkspaceRepository struct {
	db *gorm.DB
}

func NewWorkspaceRepository(db *gorm.DB) IWorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

func (r *WorkspaceRepository) WithTx(tx *gorm.DB) IWorkspaceRepository {
	return &WorkspaceRepository{db: tx}
}

func (r *WorkspaceRepository) Create(ws *model.Workspace) error {
	return r.db.Create(ws).Error
}

func (r *WorkspaceRepository) FindByID(id uuid.UUID) (*model.Workspace, error) {
	var ws model.Workspace
	if err := r.db.First(&ws, id).Error; err != nil {
		return nil, err
	}
	return &ws, nil
}

func (r *WorkspaceRepository) FindByUserID(userID uuid.UUID, page, pageSize int) ([]model.Workspace, int64, error) {
	const join = "JOIN workspace_members ON workspaces.id = workspace_members.workspace_id"
	const where = "workspace_members.user_id = ?"

	var total int64
	if err := r.db.Model(&model.Workspace{}).Joins(join).Where(where, userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	var workspaces []model.Workspace
	err := r.db.Joins(join).Where(where, userID).
		Order("workspaces.created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&workspaces).Error
	if err != nil {
		return nil, 0, err
	}
	return workspaces, total, nil
}

func (r *WorkspaceRepository) Delete(id uuid.UUID) error {
	result := r.db.Delete(&model.Workspace{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
