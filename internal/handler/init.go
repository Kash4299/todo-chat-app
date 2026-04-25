package handler

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(NewUserHandler),
	fx.Provide(NewLocalAuthHandler),
	fx.Provide(NewTaskHandler),
	fx.Provide(NewChatHandler),
)
