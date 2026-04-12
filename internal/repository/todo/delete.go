package todo

import "github.com/Kash4299/todo-chat-app/internal/model"

func (r *TodoRepository) Delete(id uint) error {
	return r.db.Delete(&model.Todo{}, id).Error
}
