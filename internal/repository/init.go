package repository

import (
	"github.com/Kash4299/todo-chat-app/internal/repository/chat"
	"github.com/Kash4299/todo-chat-app/internal/repository/todo"
	"github.com/Kash4299/todo-chat-app/internal/repository/user"
	"go.uber.org/fx"
)

var Module = fx.Module("repository",
	fx.Provide(
		fx.Annotate(user.NewUserRepository, fx.As(new(user.IUserRepository))),
		fx.Annotate(todo.NewTodoRepository, fx.As(new(todo.ITodoRepository))),
		fx.Annotate(chat.NewChatRepository, fx.As(new(chat.IChatRepository))),
	),
)
