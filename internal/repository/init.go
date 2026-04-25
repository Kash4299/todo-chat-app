package repository

import (
	"github.com/Kash4299/todo-chat-app/internal/repository/message"
	"github.com/Kash4299/todo-chat-app/internal/repository/task"
	"github.com/Kash4299/todo-chat-app/internal/repository/taskmember"
	"github.com/Kash4299/todo-chat-app/internal/repository/token"
	"github.com/Kash4299/todo-chat-app/internal/repository/user"
	"github.com/Kash4299/todo-chat-app/internal/repository/useridentity"
	"github.com/Kash4299/todo-chat-app/internal/repository/workspacemember"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(user.NewUserRepository),
	fx.Provide(task.NewTaskRepository),
	fx.Provide(taskmember.NewTaskMemberRepository),
	fx.Provide(message.NewMessageRepository),
	fx.Provide(token.NewRefreshTokenRepository),
	fx.Provide(useridentity.NewUserIdentityRepository),
	fx.Provide(workspacemember.NewWorkspaceMemberRepository),
)
