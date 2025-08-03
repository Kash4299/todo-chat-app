package middleware

import (
	"go.uber.org/fx"
)

// Module provides all middleware dependencies
var Module = fx.Options(
// Add middleware providers here when needed
// fx.Provide(NewAuthMiddleware),
// fx.Provide(NewRateLimitMiddleware),
)
