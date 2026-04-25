package route

import (
	"github.com/Kash4299/todo-chat-app/internal/handler"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, userHandler *handler.UserHandler, taskHandler *handler.TaskHandler, chatHandler *handler.ChatHandler, authMiddleware *middleware.AuthMiddleware) {
	// Root ping
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	api := r.Group("/api/v1")
	{
		// Auth0 sync webhooks — protected via shared secret in the handler
		api.POST("/webhooks/auth0/sync", userHandler.SyncAuth0Webhook)

		authenticated := api.Group("")
		authenticated.Use(authMiddleware.Handle())
		{
			tasks := authenticated.Group("/tasks")
			{
				tasks.POST("", taskHandler.Create)
				tasks.GET("/:id", taskHandler.GetByID)
			}

			ws := authenticated.Group("/ws")
			{
				ws.GET("/chat/:taskID", chatHandler.HandleWebSocket)
			}
		}
	}
}
