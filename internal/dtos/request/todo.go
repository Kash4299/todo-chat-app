package request

import (
	"todo/internal/model"

	"github.com/google/uuid"
)

type CreateTodoRequest struct {
	Title       string             `json:"title"`
	Status      model.TodoStatus   `json:"status"`
	Priority    model.TodoPriority `json:"priority"`
	Description string             `json:"description"`
	Assigned    uuid.UUID          `json:"assigned"`
}
