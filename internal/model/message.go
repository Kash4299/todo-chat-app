package model

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	ChannelID   uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"channel_id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null"                              json:"user_id"`
	MessageType string     `gorm:"not null;default:'TEXT';size:20"                 json:"message_type"`
	Content     string     `gorm:"type:text;not null;default:''"                   json:"content"`
	ReplyToID   *uuid.UUID `gorm:"type:uuid"                                       json:"reply_to_id,omitempty"`
	IsEdited    bool       `gorm:"not null;default:false"                          json:"is_edited"`
	CreatedAt   time.Time  `                                                        json:"created_at"`
	UpdatedAt   time.Time  `                                                        json:"updated_at"`
	// SearchVector is GENERATED ALWAYS AS in DB — never write this field
}
