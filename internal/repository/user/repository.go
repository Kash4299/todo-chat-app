package user

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IUserRepository interface {
	Create(user *model.User) error
	FindByAuth0ID(auth0ID string) (*model.User, error)
	FindByID(id uuid.UUID) (*model.User, error)
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) IUserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByAuth0ID(auth0ID string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("auth0_id = ?", auth0ID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
