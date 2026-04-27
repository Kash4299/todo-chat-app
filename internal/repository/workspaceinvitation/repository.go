package workspaceinvitation

import (
	"time"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IWorkspaceInvitationRepository interface {
	WithTx(tx *gorm.DB) IWorkspaceInvitationRepository
	Create(workspaceInvitation *model.WorkspaceInvitation) error
	FindByToken(token string) (*model.WorkspaceInvitation, error)
	FindByWorkspaceAndEmail(workspaceID uuid.UUID, email string) (*model.WorkspaceInvitation, error)
	MarkUsed(id uuid.UUID, usedAt time.Time) error
	Update(workspaceInvitation *model.WorkspaceInvitation) error
	ListPendingByWorkspace(workspaceID uuid.UUID) ([]model.WorkspaceInvitation, error)
}

type WorkspaceInvitationRepository struct {
	db *gorm.DB
}

func NewWorkspaceInvitationRepository(db *gorm.DB) IWorkspaceInvitationRepository {
	return &WorkspaceInvitationRepository{db: db}
}

func (r *WorkspaceInvitationRepository) WithTx(tx *gorm.DB) IWorkspaceInvitationRepository {
	return &WorkspaceInvitationRepository{db: tx}
}

func (r *WorkspaceInvitationRepository) Create(workspaceInvitation *model.WorkspaceInvitation) error {
	return r.db.Create(workspaceInvitation).Error
}

func (r *WorkspaceInvitationRepository) FindByToken(token string) (*model.WorkspaceInvitation, error) {
	var workspaceInvitation model.WorkspaceInvitation
	if err := r.db.Where("token = ?", token).First(&workspaceInvitation).Error; err != nil {
		return nil, err
	}

	return &workspaceInvitation, nil
}

func (r *WorkspaceInvitationRepository) FindByWorkspaceAndEmail(workspaceID uuid.UUID, email string) (*model.WorkspaceInvitation, error) {
	var workspaceInvitation model.WorkspaceInvitation
	if err := r.db.Where("workspace_id = ? AND email = ?", workspaceID, email).First(&workspaceInvitation).Error; err != nil {
		return nil, err
	}

	return &workspaceInvitation, nil
}

func (r *WorkspaceInvitationRepository) Update(workspaceInvitation *model.WorkspaceInvitation) error {
	return r.db.Save(workspaceInvitation).Error
}

func (r *WorkspaceInvitationRepository) MarkUsed(id uuid.UUID, usedAt time.Time) error {
	result := r.db.Model(&model.WorkspaceInvitation{}).Where("id = ?", id).Updates(map[string]any{"used_at": usedAt})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *WorkspaceInvitationRepository) ListPendingByWorkspace(workspaceID uuid.UUID) ([]model.WorkspaceInvitation, error) {
	var invitations []model.WorkspaceInvitation
	if err := r.db.Where("workspace_id = ? AND used_at IS NULL", workspaceID).Order("created_at DESC").Find(&invitations).Error; err != nil {
		return nil, err
	}

	return invitations, nil
}
