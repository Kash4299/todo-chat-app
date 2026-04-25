package model

import (
	"time"

	"github.com/google/uuid"
)

type Reaction struct {
	MessageID uuid.UUID `gorm:"type:uuid;primaryKey" json:"message_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	Emoji     string    `gorm:"primaryKey;size:10"   json:"emoji"`
	CreatedAt time.Time `gorm:"autoCreateTime"       json:"created_at"`
}
