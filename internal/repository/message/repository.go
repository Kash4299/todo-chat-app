package message

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IMessageRepository interface {
	Save(message *model.Message) error
	// FindByChannelID returns messages before beforeID (cursor pagination), newest first.
	// Pass nil beforeID to get the latest messages.
	FindByChannelID(channelID uuid.UUID, limit int, beforeID *uuid.UUID) ([]model.Message, error)
	Search(channelID uuid.UUID, query string, limit int) ([]model.Message, error)
}

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) IMessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Save(message *model.Message) error {
	return r.db.Create(message).Error
}

func (r *MessageRepository) FindByChannelID(channelID uuid.UUID, limit int, beforeID *uuid.UUID) ([]model.Message, error) {
	var messages []model.Message
	q := r.db.Where("channel_id = ?", channelID).Order("created_at DESC").Limit(limit)
	if beforeID != nil {
		// Cursor: fetch messages older than the given message
		var cursor model.Message
		if err := r.db.Select("created_at").First(&cursor, beforeID).Error; err != nil {
			return nil, err
		}
		q = q.Where("created_at < ?", cursor.CreatedAt)
	}
	if err := q.Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *MessageRepository) Search(channelID uuid.UUID, query string, limit int) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.
		Where("channel_id = ? AND search_vector @@ plainto_tsquery('simple', unaccent(?))", channelID, query).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}
