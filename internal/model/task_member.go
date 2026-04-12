package model

import (
	"time"

	"github.com/google/uuid"
)

type TaskMember struct {
	TaskID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"task_id"`
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id"`
	Role     string    `gorm:"default:'MEMBER';size:50" json:"role"`
	JoinedAt time.Time `gorm:"autoCreateTime" json:"joined_at"`
}
