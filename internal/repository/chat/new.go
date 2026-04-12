package chat

import "gorm.io/gorm"

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) IChatRepository {
	return &ChatRepository{db: db}
}
