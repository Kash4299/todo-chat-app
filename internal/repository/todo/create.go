package todo

import "github.com/Kash4299/todo-chat-app/internal/model"

func (r *TodoRepository) Create(todo *model.Todo) error {
	return r.db.Create(todo).Error
}
