package todo

import "github.com/Kash4299/todo-chat-app/internal/model"

type ITodoRepository interface {
	Create(todo *model.Todo) error
	FindByID(id uint) (*model.Todo, error)
	FindByUserID(userID uint) ([]model.Todo, error)
	Update(todo *model.Todo) error
	Delete(id uint) error
}
