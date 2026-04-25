package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Email       string    `gorm:"uniqueIndex;not null;size:255"                   json:"email"`
	DisplayName string    `gorm:"not null;size:100"                               json:"display_name"`
	AvatarURL   string    `gorm:"size:500"                                        json:"avatar_url,omitempty"`
	StatusText  string    `gorm:"not null;default:'';size:150"                    json:"status_text"`
	IsActive     bool      `gorm:"not null;default:true"  json:"is_active"`
	PasswordHash *string  `gorm:"size:255"               json:"-"`
	CreatedAt    time.Time `                              json:"created_at"`
	UpdatedAt    time.Time `                              json:"updated_at"`
}
