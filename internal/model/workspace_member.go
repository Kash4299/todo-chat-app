package model

import (
	"time"

	"github.com/google/uuid"
)

type WorkspaceMember struct {
	WorkspaceID uuid.UUID `gorm:"type:uuid;primaryKey"              json:"workspace_id"`
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey;index"        json:"user_id"`
	Role        string    `gorm:"not null;default:'MEMBER';size:20" json:"role"`
	JoinedAt    time.Time `gorm:"autoCreateTime"                    json:"joined_at"`
}
