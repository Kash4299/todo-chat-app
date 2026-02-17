package repositories

import (
	"todo/internal/repositories/chat"
	"todo/internal/repositories/todo"
	"todo/internal/repositories/user"

	"go.uber.org/fx"
)

// Module provides all repository dependencies
var Module = fx.Options(
	fx.Provide(
		todo.NewTodoRepository,
		user.NewUserRepository,
		chat.NewChatRepository,
	),
)
