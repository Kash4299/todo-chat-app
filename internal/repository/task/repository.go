package task

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ITaskRepository interface {
	Create(task *model.Task) error
	FindByID(id uuid.UUID) (*model.Task, error)
	FindByWorkspace(workspaceID uuid.UUID, offset, limit int) ([]model.Task, error)
	CountByWorkspace(workspaceID uuid.UUID) (int64, error)
	Update(task *model.Task) error
	Delete(id uuid.UUID) error
}

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) ITaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(task *model.Task) error {
	return r.db.Create(task).Error
}

func (r *TaskRepository) FindByID(id uuid.UUID) (*model.Task, error) {
	var task model.Task
	if err := r.db.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) FindByWorkspace(workspaceID uuid.UUID, offset, limit int) ([]model.Task, error) {
	var tasks []model.Task
	if err := r.db.Where("workspace_id = ?", workspaceID).
		Order("position ASC, created_at DESC").
		Offset(offset).Limit(limit).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *TaskRepository) CountByWorkspace(workspaceID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.Model(&model.Task{}).Where("workspace_id = ?", workspaceID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *TaskRepository) Update(task *model.Task) error {
	return r.db.Save(task).Error
}

func (r *TaskRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.Task{}, id).Error
}
