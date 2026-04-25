package useridentity

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IUserIdentityRepository interface {
	Create(identity *model.UserIdentity) error
	FindByProviderSubject(providerSubject string) (*model.UserIdentity, error)
	FindByUserID(userID uuid.UUID) ([]model.UserIdentity, error)
}

type UserIdentityRepository struct {
	db *gorm.DB
}

func NewUserIdentityRepository(db *gorm.DB) IUserIdentityRepository {
	return &UserIdentityRepository{db: db}
}

func (r *UserIdentityRepository) Create(identity *model.UserIdentity) error {
	return r.db.Create(identity).Error
}

func (r *UserIdentityRepository) FindByProviderSubject(providerSubject string) (*model.UserIdentity, error) {
	var identity model.UserIdentity
	if err := r.db.Where("provider_subject = ?", providerSubject).First(&identity).Error; err != nil {
		return nil, err
	}
	return &identity, nil
}

func (r *UserIdentityRepository) FindByUserID(userID uuid.UUID) ([]model.UserIdentity, error) {
	var identities []model.UserIdentity
	if err := r.db.Where("user_id = ?", userID).Find(&identities).Error; err != nil {
		return nil, err
	}
	return identities, nil
}
