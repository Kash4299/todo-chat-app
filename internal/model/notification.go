package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Notification struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"user_id"`
	Type      string         `gorm:"not null;size:50"                                json:"type"`
	Payload   datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"                json:"payload"`
	IsRead    bool           `gorm:"not null;default:false"                          json:"is_read"`
	CreatedAt time.Time      `                                                        json:"created_at"`
}
