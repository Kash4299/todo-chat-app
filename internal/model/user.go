package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Auth0ID     string    `gorm:"uniqueIndex;not null;size:255" json:"auth0_id"`
	Email       string    `gorm:"uniqueIndex;not null;size:255" json:"email"`
	DisplayName string    `gorm:"not null;size:100" json:"display_name"`
	AvatarURL   string    `gorm:"size:500" json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
}
