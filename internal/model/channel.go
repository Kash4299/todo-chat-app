package model

import (
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index"                        json:"workspace_id"`
	Name        string    `gorm:"not null;size:100"                               json:"name"`
	Type        string    `gorm:"not null;default:'PUBLIC';size:20"               json:"type"`
	CreatedBy   uuid.UUID `gorm:"type:uuid;not null"                              json:"created_by"`
	CreatedAt   time.Time `                                                        json:"created_at"`
}
