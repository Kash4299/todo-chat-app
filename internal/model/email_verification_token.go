package model

import (
	"time"

	"github.com/google/uuid"
)

type EmailVerificationToken struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"uniqueIndex;not null;size:64"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}
