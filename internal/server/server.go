package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/handler"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/route"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func NewGinEngine(cfg *config.Config) *gin.Engine {
	gin.SetMode(ginMode(cfg.GinMode))

	r := gin.New()
	r.Use(gin.Recovery()) // keep panic recovery; structured request logging lands in T87
	r.Use(gin.Logger())
	r.Use(middleware.CORS(splitOrigins(cfg.AllowedOrigins)))
	return r
}

func ginMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "debug":
		return gin.DebugMode
	case "test":
		return gin.TestMode
	default:
		return gin.ReleaseMode
	}
}

func splitOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

func StartServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	router *gin.Engine,
	userHandler *handler.UserHandler,
	localAuthHandler *handler.LocalAuthHandler,
	taskHandler *handler.TaskHandler,
	chatHandler *handler.ChatHandler,
	authMiddleware *middleware.AuthMiddleware,
	workspaceHandler *handler.WorkspaceHandler,
	workspaceInvitationHandler *handler.WorkspaceInvitationHandler,
	healthHandler *handler.HealthHandler,
	rbacMiddleware *middleware.RBACMiddleware,
) {
	route.SetupRoutes(router, userHandler, localAuthHandler, taskHandler, chatHandler, workspaceHandler, workspaceInvitationHandler, healthHandler, authMiddleware, rbacMiddleware)

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
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return fmt.Errorf("listen on %s: %w", srv.Addr, err)
			}

			go func() {
				log.Printf("server starting on port %s", cfg.ServerPort)
				if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
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
