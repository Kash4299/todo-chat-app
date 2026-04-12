package server

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/handler"
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
	todoHandler *handler.TodoHandler,
	chatHandler *handler.ChatHandler,
) {
	route.RegisterRoutes(router, userHandler, todoHandler, chatHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: router,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Printf("server starting on port %s", cfg.ServerPort)
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatalf("server failed: %v", err)
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
