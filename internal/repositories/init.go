package repositories

import (
	"todo/internal/repositories/todo"

	"go.uber.org/fx"
)

// Module provides all repository dependencies
var Module = fx.Options(
	fx.Provide(
		todo.NewTodoRepository,
	),
)
