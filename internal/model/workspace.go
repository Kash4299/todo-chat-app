package model

import (
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Name      string    `gorm:"not null;size:100"                               json:"name"`
	Slug      string    `gorm:"uniqueIndex;not null;size:100"                   json:"slug"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null;index"                        json:"owner_id"`
	CreatedAt time.Time `                                                        json:"created_at"`
	UpdatedAt time.Time `                                                        json:"updated_at"`
}
