package server

import (
	"todo/internal/handlers/todo"
	"todo/internal/handlers/user"

	"go.uber.org/fx"
)

type Handlers struct {
	TodoHandler *todo.TodoHandler
	UserHandler *user.UserHandler
}

func NewHandlers(
	todoHandler *todo.TodoHandler,
	userHandler *user.UserHandler,
) Handlers {
	return Handlers{
		TodoHandler: todoHandler,
		UserHandler: userHandler,
	}
}

// HandlersModule provides the Handlers struct through fx
var HandlersModule = fx.Options(
	fx.Provide(NewHandlers),
)
