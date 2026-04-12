package user

import "github.com/Kash4299/todo-chat-app/internal/model"

type IUserRepository interface {
	Create(user *model.User) error
	FindByID(id uint) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
}
