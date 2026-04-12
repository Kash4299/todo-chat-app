package user

import "github.com/Kash4299/todo-chat-app/internal/model"

func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}
