package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"todo/internal/config"
	"todo/internal/handlers/todo"
	"todo/internal/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	config *config.Config
	logger *zap.Logger
	engine *gin.Engine
	server *http.Server
}

// NewServer creates a new server instance with all dependencies
func NewServer(
	config *config.Config,
	logger *zap.Logger,
	todoHandler *todo.TodoHandler,
) *Server {
	// Set Gin mode based on environment
	if config.Server.Host == "localhost" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin engine
	engine := gin.New()

	// Add middleware
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORSWithDefaults())
	engine.Use(loggerMiddleware(logger))

	// Setup routes using the router
	router := NewRouter(engine, todoHandler)
	router.SetupRoutes()

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port),
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		config: config,
		logger: logger,
		engine: engine,
		server: server,
	}
}



// loggerMiddleware adds request logging to all routes
func loggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.Info("HTTP Request",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		)
		return ""
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server",
		zap.String("host", s.config.Server.Host),
		zap.String("port", s.config.Server.Port),
	)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server...")
	return s.server.Shutdown(ctx)
}

// GetEngine returns the Gin engine (useful for testing)
func (s *Server) GetEngine() *gin.Engine {
	return s.engine
}

// GetServer returns the HTTP server (useful for testing)
func (s *Server) GetServer() *http.Server {
	return s.server
}
