package services

import (
	"go.uber.org/fx"
)

// Module provides all service dependencies
var Module = fx.Options(
	fx.Provide(
		NewHub,
		NewTodoService,
		NewUserService,
		NewChatService,
	),
)
