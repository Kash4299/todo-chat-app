package todo

import "github.com/Kash4299/todo-chat-app/internal/model"

func (r *TodoRepository) FindByID(id uint) (*model.Todo, error) {
	var todo model.Todo
	if err := r.db.First(&todo, id).Error; err != nil {
		return nil, err
	}
	return &todo, nil
}

func (r *TodoRepository) FindByUserID(userID uint) ([]model.Todo, error) {
	var todos []model.Todo
	if err := r.db.Where("user_id = ?", userID).Find(&todos).Error; err != nil {
		return nil, err
	}
	return todos, nil
}
