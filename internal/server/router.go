package server

import (
	"net/http"
	"time"

	"todo/common"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine   *gin.Engine
	handlers Handlers
}

func NewRouter(engine *gin.Engine, handlers Handlers) *Router {
	return &Router{
		engine:   engine,
		handlers: handlers,
	}
}

func (r *Router) SetupRoutes() {
	r.engine.GET("/health", r.healthCheck)

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		r.setupTodoRoutes(v1)
	}

	// 404 handler
	r.engine.NoRoute(r.notFoundHandler)
}

func (r *Router) healthCheck(c *gin.Context) {
	response := common.SuccessResponseWithMessage("Server is running", gin.H{
		"time": time.Now().UTC(),
	})
	c.JSON(http.StatusOK, response)
}

func (r *Router) setupTodoRoutes(v1 *gin.RouterGroup) {
	todos := v1.Group("/todos")
	{
		todos.GET("", r.handlers.TodoHandler.GetAll)
		todos.GET("/:id", r.handlers.TodoHandler.GetByID)
		todos.POST("", r.handlers.TodoHandler.Create)
		todos.PUT("/:id", r.handlers.TodoHandler.Update)
		todos.DELETE("/:id", r.handlers.TodoHandler.Delete)
	}

	users := v1.Group("/users")
	{
		// users.GET("/:id", r.handlers.TodoHandler.GetByID)
		users.POST("", r.handlers.UserHandler.CreateUser)
		// users.PUT("/:id", r.handlers.TodoHandler.Update)
		// users.DELETE("/:id", r.handlers.TodoHandler.Delete)
	}
}

// notFoundHandler handles 404 responses
func (r *Router) notFoundHandler(c *gin.Context) {
	response := common.NotFoundResponse("The requested resource was not found")
	c.JSON(http.StatusNotFound, response)
}
