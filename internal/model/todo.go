package model

import "github.com/google/uuid"

type TodoStatus string

const (
	Backlog    TodoStatus = "backlog"
	ToDo       TodoStatus = "todo"
	InProgress TodoStatus = "in_progress"
	Testing    TodoStatus = "testing"
	Done       TodoStatus = "done"
)

type TodoPriority int

const (
	Normal TodoPriority = 0
	Medium TodoPriority = 1
	High   TodoPriority = 2
	Urgent TodoPriority = 3
)

type Todo struct {
	BaseModel

	Title       string       `gorm:"column:title" json:"title"`
	Status      TodoStatus   `gorm:"column:status" json:"status"`
	Priority    TodoPriority `gorm:"column:priority" json:"priority"`
	Description string       `gorm:"column:description" json:"description"`
	CreatedBy   uuid.UUID    `gorm:"column:created_by" json:"created_by"`
	Assigned    uuid.UUID    `gorm:"column:assigned" json:"assigned"`
}
