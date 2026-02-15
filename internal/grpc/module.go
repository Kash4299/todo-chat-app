package grpcserver

import (
	"go.uber.org/fx"
)

// Module provides all gRPC dependencies through fx
var Module = fx.Options(
	fx.Provide(
		NewTodoGRPCHandler,
		NewUserGRPCHandler,
		NewServer,
	),
)
