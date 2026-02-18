package repositories

import (
	"todo/internal/repositories/chat"
	"todo/internal/repositories/todo"
	"todo/internal/repositories/user"

	"go.uber.org/fx"
)

// Module provides all repository dependencies.
// fx.As() tells Fx that concrete types satisfy the corresponding interfaces,
// so that Hub (which depends on interfaces) can receive them.
var Module = fx.Options(
	fx.Provide(
		fx.Annotate(todo.NewTodoRepository, fx.As(new(todo.ITodoRepository))),
		fx.Annotate(user.NewUserRepository, fx.As(new(user.IUserRepository))),
		fx.Annotate(chat.NewChatRepository, fx.As(new(chat.IChatRepository))),
	),
)
