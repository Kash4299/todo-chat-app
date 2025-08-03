package todo

import (
	"todo/internal/database"
)

type TodoRepository struct {
	db *database.Database
}

func NewTodoRepository(db *database.Database) *TodoRepository {
	return &TodoRepository{db: db}
}
