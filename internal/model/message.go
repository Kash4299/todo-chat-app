package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Message struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TaskID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_messages_task_id_created_at" json:"task_id"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	MessageType string         `gorm:"default:'TEXT';size:20" json:"message_type"`
	Content     string         `gorm:"type:text" json:"content"`
	Attachments datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"attachments"`
	CreatedAt   time.Time      `gorm:"index:idx_messages_task_id_created_at,sort:desc" json:"created_at"`
	// SearchVector is omitted from struct mapping writes as it is GENERATED ALWAYS AS
}
