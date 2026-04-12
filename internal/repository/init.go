package repository

import (
	"github.com/Kash4299/todo-chat-app/internal/repository/message"
	"github.com/Kash4299/todo-chat-app/internal/repository/task"
	"github.com/Kash4299/todo-chat-app/internal/repository/taskmember"
	"github.com/Kash4299/todo-chat-app/internal/repository/user"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(user.NewUserRepository),
	fx.Provide(task.NewTaskRepository),
	fx.Provide(taskmember.NewTaskMemberRepository),
	fx.Provide(message.NewMessageRepository),
)
