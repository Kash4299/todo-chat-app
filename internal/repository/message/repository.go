package message

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IMessageRepository interface {
	SaveMessage(message *model.Message) error
	GetMessagesByTask(taskID uuid.UUID, limit int) ([]model.Message, error)
}

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) IMessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) SaveMessage(message *model.Message) error {
	return r.db.Create(message).Error
}

func (r *MessageRepository) GetMessagesByTask(taskID uuid.UUID, limit int) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.Where("task_id = ?", taskID).Order("created_at desc").Limit(limit).Find(&messages).Error
	return messages, err
}
