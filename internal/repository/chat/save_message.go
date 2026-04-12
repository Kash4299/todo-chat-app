package chat

import "github.com/Kash4299/todo-chat-app/internal/model"

func (r *ChatRepository) SaveMessage(msg *model.ChatMessage) error {
	return r.db.Create(msg).Error
}

func (r *ChatRepository) GetMessagesByRoomID(roomID string) ([]model.ChatMessage, error) {
	var messages []model.ChatMessage
	if err := r.db.Where("room_id = ?", roomID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}
