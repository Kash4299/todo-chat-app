package chat

import (
	"context"
	"todo/internal/model"
)

// SaveMessage persists a chat message to the database
func (r *ChatRepository) SaveMessage(ctx context.Context, msg *model.ChatMessage) error {
	return r.db.DB.WithContext(ctx).Create(msg).Error
}

// GetMessagesByRoom retrieves the latest messages for a room, ordered by creation time
func (r *ChatRepository) GetMessagesByRoom(ctx context.Context, room string, limit int) (*[]model.ChatMessage, error) {
	var messages []model.ChatMessage
	err := r.db.DB.WithContext(ctx).
		Where("room = ?", room).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return &messages, nil
}
