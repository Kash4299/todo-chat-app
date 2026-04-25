package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/handler"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/route"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func NewGinEngine() *gin.Engine {
	r := gin.Default()
	return r
}

func StartServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	router *gin.Engine,
	userHandler *handler.UserHandler,
	taskHandler *handler.TaskHandler,
	chatHandler *handler.ChatHandler,
	authMiddleware *middleware.AuthMiddleware,
) {
	route.SetupRoutes(router, userHandler, taskHandler, chatHandler, authMiddleware)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Printf("server starting on port %s", cfg.ServerPort)
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("server failed: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("server shutting down...")
			return srv.Shutdown(ctx)
		},
	})
}
