package taskmember

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ITaskMemberRepository interface {
	AddMember(member *model.TaskMember) error
	RemoveMember(taskID, userID uuid.UUID) error
	IsMember(taskID, userID uuid.UUID) (bool, error)
	GetMembersByTask(taskID uuid.UUID) ([]model.TaskMember, error)
	GetTasksByUser(userID uuid.UUID) ([]model.TaskMember, error)
}

type TaskMemberRepository struct {
	db *gorm.DB
}

func NewTaskMemberRepository(db *gorm.DB) ITaskMemberRepository {
	return &TaskMemberRepository{db: db}
}

func (r *TaskMemberRepository) AddMember(member *model.TaskMember) error {
	return r.db.Create(member).Error
}

func (r *TaskMemberRepository) RemoveMember(taskID, userID uuid.UUID) error {
	return r.db.Where("task_id = ? AND user_id = ?", taskID, userID).Delete(&model.TaskMember{}).Error
}

func (r *TaskMemberRepository) IsMember(taskID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&model.TaskMember{}).Where("task_id = ? AND user_id = ?", taskID, userID).Count(&count).Error
	return count > 0, err
}

func (r *TaskMemberRepository) GetMembersByTask(taskID uuid.UUID) ([]model.TaskMember, error) {
	var members []model.TaskMember
	err := r.db.Where("task_id = ?", taskID).Find(&members).Error
	return members, err
}

func (r *TaskMemberRepository) GetTasksByUser(userID uuid.UUID) ([]model.TaskMember, error) {
	var members []model.TaskMember
	err := r.db.Where("user_id = ?", userID).Find(&members).Error
	return members, err
}
