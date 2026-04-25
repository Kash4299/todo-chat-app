package main

import (
	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/handler"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/repository"
	"github.com/Kash4299/todo-chat-app/internal/server"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/Kash4299/todo-chat-app/pkg/database"
	"github.com/Kash4299/todo-chat-app/pkg/kafka"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		// Config
		fx.Provide(config.NewConfig),

		// Infrastructure
		fx.Provide(
			database.NewPostgresDB,
			database.NewRedisClient,
			kafka.NewKafkaClient,
			server.NewGinEngine,
		),

		// Application layers
		repository.Module,
		service.Module,
		handler.Module,
		fx.Provide(middleware.NewAuthMiddleware),

		// Server lifecycle
		fx.Invoke(server.StartServer),
	).Run()
}
