package model

import "time"

type ChatMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RoomID    string    `gorm:"index;not null;size:100" json:"room_id"`
	SenderID  uint      `gorm:"index;not null" json:"sender_id"`
	Content   string    `gorm:"not null;size:2000" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
