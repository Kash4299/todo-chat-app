package handlers

import (
	"todo/internal/handlers/todo"
	"todo/internal/handlers/user"

	"go.uber.org/fx"
)

// Module provides all handler dependencies
var Module = fx.Options(
	fx.Provide(
		todo.NewTodoHandler,
		user.NewUserHandler,
	),
)
