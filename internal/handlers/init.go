package handlers

import (
	"todo/internal/handlers/todo"

	"go.uber.org/fx"
)

// Module provides all handler dependencies
var Module = fx.Options(
	fx.Provide(
		todo.NewTodoHandler,
	),
)
