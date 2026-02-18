package main

import (
	"context"
	"log"

	"todo/internal/config"
	"todo/internal/database"
	grpcserver "todo/internal/grpc"
	"todo/internal/kafka"
	"todo/internal/redis"
	"todo/internal/repositories"
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
		redis.Module,
		kafka.Module,
		grpcserver.Module,
		// App hooks
		fx.Invoke(registerAppHooks),
	)

	app.Run()
}

// registerAppHooks registers lifecycle hooks for the application
func registerAppHooks(
	lifecycle fx.Lifecycle,
	db *database.Database,
	redisClient *redis.Client,
	kafkaClient *kafka.Client,
	grpcSrv *grpcserver.Server,
	logger *zap.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			// Start gRPC server
			go func() {
				if err := grpcSrv.Start(); err != nil {
					logger.Fatal("Failed to start gRPC server", zap.Error(err))
				}
			}()

			logger.Info("Application started successfully (gRPC only)")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down application...")

			// 1. Close database connection
			if err := db.Close(); err != nil {
				logger.Error("Error closing database connection", zap.Error(err))
			}
			logger.Info("Database connection closed")

			// 2. Close Redis connection
			if redisClient.IsEnabled() {
				if err := redisClient.Close(); err != nil {
					logger.Error("Error closing Redis connection", zap.Error(err))
				}
				logger.Info("Redis connection closed")
			}

			// 3. Close Kafka connection
			if kafkaClient.IsEnabled() {
				if err := kafkaClient.Close(); err != nil {
					logger.Error("Error closing Kafka connection", zap.Error(err))
				}
				logger.Info("Kafka connection closed")
			}

			// 4. Stop gRPC server gracefully
			grpcSrv.GracefulStop()

			logger.Info("Application shutdown completed")
			return nil
		},
	})
}
