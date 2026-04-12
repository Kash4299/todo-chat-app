package todo

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
)

func (r *TodoRepository) Update(todo *model.Todo) error {
	return r.db.Save(todo).Error
}
