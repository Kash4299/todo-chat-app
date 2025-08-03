package server

import (
	"net/http"
	"time"

	"todo/common"
	"todo/internal/handlers/todo"

	"github.com/gin-gonic/gin"
)

// Router handles all route configuration
type Router struct {
	engine      *gin.Engine
	todoHandler *todo.TodoHandler
}

// NewRouter creates a new router instance
func NewRouter(engine *gin.Engine, todoHandler *todo.TodoHandler) *Router {
	return &Router{
		engine:      engine,
		todoHandler: todoHandler,
	}
}

// SetupRoutes configures all the routes for the application
func (r *Router) SetupRoutes() {
	// Health check endpoint
	r.engine.GET("/health", r.healthCheck)

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Todo routes
		r.setupTodoRoutes(v1)
		
		// Add more route groups here as needed
		// Example: r.setupUserRoutes(v1)
		// Example: r.setupAuthRoutes(v1)
	}

	// 404 handler
	r.engine.NoRoute(r.notFoundHandler)
}

// healthCheck handles the health check endpoint
func (r *Router) healthCheck(c *gin.Context) {
	response := common.SuccessResponseWithMessage("Server is running", gin.H{
		"time": time.Now().UTC(),
	})
	c.JSON(http.StatusOK, response)
}

// setupTodoRoutes configures todo-related routes
func (r *Router) setupTodoRoutes(v1 *gin.RouterGroup) {
	todos := v1.Group("/todos")
	{
		todos.GET("", r.todoHandler.GetAll)
		todos.GET("/:id", r.todoHandler.GetByID)
		todos.POST("", r.todoHandler.Create)
		todos.PUT("/:id", r.todoHandler.Update)
		todos.DELETE("/:id", r.todoHandler.Delete)
	}
}

// notFoundHandler handles 404 responses
func (r *Router) notFoundHandler(c *gin.Context) {
	response := common.NotFoundResponse("The requested resource was not found")
	c.JSON(http.StatusNotFound, response)
}

// Example methods for future route groups (uncomment and implement as needed)

// setupUserRoutes configures user-related routes
// func (r *Router) setupUserRoutes(v1 *gin.RouterGroup) {
// 	users := v1.Group("/users")
// 	{
// 		users.GET("", r.userHandler.GetAll)
// 		users.GET("/:id", r.userHandler.GetByID)
// 		users.POST("", r.userHandler.Create)
// 		users.PUT("/:id", r.userHandler.Update)
// 		users.DELETE("/:id", r.userHandler.Delete)
// 	}
// }

// setupAuthRoutes configures authentication-related routes
// func (r *Router) setupAuthRoutes(v1 *gin.RouterGroup) {
// 	auth := v1.Group("/auth")
// 	{
// 		auth.POST("/login", r.authHandler.Login)
// 		auth.POST("/register", r.authHandler.Register)
// 		auth.POST("/logout", r.authHandler.Logout)
// 		auth.POST("/refresh", r.authHandler.RefreshToken)
// 	}
// }

// setupChatRoutes configures chat-related routes
// func (r *Router) setupChatRoutes(v1 *gin.RouterGroup) {
// 	chat := v1.Group("/chat")
// 	{
// 		chat.GET("/ws", r.chatHandler.WebSocket)
// 		chat.GET("/messages", r.chatHandler.GetMessages)
// 		chat.POST("/messages", r.chatHandler.SendMessage)
// 	}
// } 