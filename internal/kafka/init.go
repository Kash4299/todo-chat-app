package kafka

import (
	"go.uber.org/fx"
)

// Module provides Kafka dependencies
var Module = fx.Options(
	fx.Provide(NewClient),
)
