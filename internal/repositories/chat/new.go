package chat

import "todo/internal/database"

// ChatRepository implements IChatRepository using GORM
type ChatRepository struct {
	db *database.Database
}

func NewChatRepository(db *database.Database) *ChatRepository {
	return &ChatRepository{db: db}
}
