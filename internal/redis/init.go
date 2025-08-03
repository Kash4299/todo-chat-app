package redis

import (
	"todo/internal/config"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides Redis dependencies
var Module = fx.Options(
	fx.Provide(NewClient),
)

// NewClientProvider provides a Redis client with error handling
func NewClientProvider(config *config.Config, logger *zap.Logger) (*Client, error) {
	return NewClient(config, logger)
}
