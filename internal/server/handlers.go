package server

import (
	"todo/internal/handlers/todo"

	"go.uber.org/fx"
)

type Handlers struct {
	TodoHandler *todo.TodoHandler
}

func NewHandlers(
	todoHandler *todo.TodoHandler,
) Handlers {
	return Handlers{
		TodoHandler: todoHandler,
	}
}

// HandlersModule provides the Handlers struct through fx
var HandlersModule = fx.Options(
	fx.Provide(NewHandlers),
)
