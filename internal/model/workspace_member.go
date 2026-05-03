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

// WorkspaceMemberInfo is the projection returned by member-listing queries (JOIN with users).
type WorkspaceMemberInfo struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}
