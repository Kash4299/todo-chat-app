package model

import (
	"time"

	"github.com/google/uuid"
)

type TaskActivity struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TaskID       uuid.UUID `gorm:"type:uuid;not null;index"                        json:"task_id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null"                              json:"user_id"`
	ActivityType string    `gorm:"not null;size:50"                                json:"activity_type"`
	OldValue     string    `gorm:"type:text"                                       json:"old_value,omitempty"`
	NewValue     string    `gorm:"type:text"                                       json:"new_value,omitempty"`
	CreatedAt    time.Time `                                                        json:"created_at"`
}
