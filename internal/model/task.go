package model

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Title       string    `gorm:"not null;size:255" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Status      string    `gorm:"default:'TODO';size:50;index" json:"status"`
	CreatedBy   uuid.UUID `gorm:"type:uuid;not null;index" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
