package chat

import "github.com/Kash4299/todo-chat-app/internal/model"

type IChatRepository interface {
	SaveMessage(msg *model.ChatMessage) error
	GetMessagesByRoomID(roomID string) ([]model.ChatMessage, error)
}
