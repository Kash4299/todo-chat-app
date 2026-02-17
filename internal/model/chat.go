package model

import "github.com/google/uuid"

// ChatMessage represents a chat message stored in the database
type ChatMessage struct {
	BaseModel
	Room    string    `gorm:"column:room;index" json:"room"`
	Sender  uuid.UUID `gorm:"column:sender;type:uuid" json:"sender"`
	Content string    `gorm:"column:content" json:"content"`
}
