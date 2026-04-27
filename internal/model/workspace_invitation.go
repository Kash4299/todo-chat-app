package model

import (
	"time"

	"github.com/google/uuid"
)

type WorkspaceInvitation struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	CreatedAt   time.Time  `gorm:"not null"                                        json:"created_at"`
	WorkspaceID uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"workspace_id"`
	InvitedBy   uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"invited_by"`
	Email       string     `gorm:"not null;size:255"                               json:"email"`
	Token       string     `gorm:"not null;size:64;uniqueIndex"                    json:"token"`
	ExpiresAt   time.Time  `gorm:"not null"                                        json:"expires_at"`
	UsedAt      *time.Time `json:"used_at"`
}
