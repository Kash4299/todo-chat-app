package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"todo/internal/config"
	"todo/internal/database"
	"todo/internal/handlers"
	"todo/internal/kafka"
	"todo/internal/redis"
	"todo/internal/repositories"
	"todo/internal/server"
	"todo/internal/services"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	app := fx.New(
		// Core modules
		fx.Provide(
			config.NewConfig,
			database.NewDatabase,
			zap.NewProduction,
		),
		// Feature modules
		repositories.Module,
		services.Module,
		handlers.Module,
		redis.Module,
		kafka.Module,
		fx.Provide(server.NewServer),
		// App hooks
		fx.Invoke(registerAppHooks),
	)

	app.Run()
}

// registerAppHooks registers lifecycle hooks for the application
func registerAppHooks(
	lifecycle fx.Lifecycle,
	server *server.Server,
	db *database.Database,
	redisClient *redis.Client,
	kafkaClient *kafka.Client,
	logger *zap.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			// Start server in a goroutine
			go func() {
				if err := server.Start(); err != nil && err != http.ErrServerClosed {
					logger.Fatal("Failed to start server", zap.Error(err))
				}
			}()
			logger.Info("Application started successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down application...")

			// 1. Close database connection first
			if err := db.Close(); err != nil {
				logger.Error("Error closing database connection", zap.Error(err))
				// Continue to shutdown other services even if db close fails
			}
			logger.Info("Database connection closed")

			// 2. Close Redis connection
			if redisClient.IsEnabled() {
				if err := redisClient.Close(); err != nil {
					logger.Error("Error closing Redis connection", zap.Error(err))
					// Continue to shutdown other services even if redis close fails
				}
				logger.Info("Redis connection closed")
			}

			// 3. Close Kafka connection
			if kafkaClient.IsEnabled() {
				if err := kafkaClient.Close(); err != nil {
					logger.Error("Error closing Kafka connection", zap.Error(err))
					// Continue to shutdown server even if kafka close fails
				}
				logger.Info("Kafka connection closed")
			}

			// 4. Graceful shutdown of server
			shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Error("Error during server shutdown", zap.Error(err))
				return err
			}
			logger.Info("Application shutdown completed")
			return nil
		},
	})
}
