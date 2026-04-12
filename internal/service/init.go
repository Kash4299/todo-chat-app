package service

import "go.uber.org/fx"

var Module = fx.Module("service",
	fx.Provide(
		fx.Annotate(NewUserService, fx.As(new(IUserService))),
		fx.Annotate(NewTodoService, fx.As(new(ITodoService))),
		fx.Annotate(NewChatService, fx.As(new(IChatService))),
	),
)
