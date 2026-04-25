package model

import (
	"time"

	"github.com/google/uuid"
)

// ChannelMember tracks which users belong to a channel.
// Replaces the old TaskMember concept — channels are now the unit of membership.
type ChannelMember struct {
	ChannelID  uuid.UUID `gorm:"type:uuid;primaryKey"       json:"channel_id"`
	UserID     uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id"`
	LastReadAt time.Time `gorm:"autoCreateTime"             json:"last_read_at"`
	JoinedAt   time.Time `gorm:"autoCreateTime"             json:"joined_at"`
}
