package service

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(NewUserService),
	fx.Provide(NewTaskService),
	fx.Provide(NewChatService),
)
