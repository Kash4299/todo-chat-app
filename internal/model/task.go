package model

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	WorkspaceID  uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"workspace_id"`
	ChannelID    *uuid.UUID `gorm:"type:uuid"                                       json:"channel_id,omitempty"`
	Title        string     `gorm:"not null;size:255"                               json:"title"`
	Description  string     `gorm:"type:text;not null;default:''"                   json:"description"`
	Status       string     `gorm:"not null;default:'TODO';size:20"                 json:"status"`
	Priority     string     `gorm:"not null;default:'MEDIUM';size:20"               json:"priority"`
	AssigneeID   *uuid.UUID `gorm:"type:uuid"                                       json:"assignee_id,omitempty"`
	CreatedBy    uuid.UUID  `gorm:"type:uuid;not null"                              json:"created_by"`
	ParentTaskID *uuid.UUID `gorm:"type:uuid"                                       json:"parent_task_id,omitempty"`
	Position     int        `gorm:"not null;default:0"                              json:"position"`
	DueDate      *time.Time `                                                        json:"due_date,omitempty"`
	CreatedAt    time.Time  `                                                        json:"created_at"`
	UpdatedAt    time.Time  `                                                        json:"updated_at"`
}
