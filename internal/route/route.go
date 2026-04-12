package route

import (
	"github.com/Kash4299/todo-chat-app/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	router *gin.Engine,
	userHandler *handler.UserHandler,
	todoHandler *handler.TodoHandler,
	chatHandler *handler.ChatHandler,
) {
	api := router.Group("/api/v1")
	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
		}

		// Todo routes
		todos := api.Group("/todos")
		{
			todos.POST("", todoHandler.Create)
			todos.GET("/:id", todoHandler.GetByID)
			todos.GET("/user/:user_id", todoHandler.GetByUserID)
			todos.PUT("/:id", todoHandler.Update)
			todos.DELETE("/:id", todoHandler.Delete)
		}

		// Chat routes
		chat := api.Group("/chat")
		{
			chat.GET("/messages/:roomID", chatHandler.GetMessages)
			chat.POST("/messages/:roomID", chatHandler.SendMessage)
		}
	}

	// WebSocket route
	router.GET("/ws/chat/:roomID", chatHandler.HandleWebSocket)
}
