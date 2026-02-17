package chat

import (
	"context"
	"todo/internal/model"
)

// IChatRepository defines the contract for chat message persistence
type IChatRepository interface {
	SaveMessage(ctx context.Context, msg *model.ChatMessage) error
	GetMessagesByRoom(ctx context.Context, room string, limit int) (*[]model.ChatMessage, error)
}
