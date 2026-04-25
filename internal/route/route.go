package route

import (
	"github.com/Kash4299/todo-chat-app/internal/handler"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	r *gin.Engine,
	userHandler *handler.UserHandler,
	taskHandler *handler.TaskHandler,
	chatHandler *handler.ChatHandler,
	authMiddleware *middleware.AuthMiddleware,
) {
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	api := r.Group("/api/v1")

	// Public webhook endpoints
	webhooks := api.Group("/webhooks")
	{
		webhooks.POST("/auth0/users.sync", userHandler.SyncAuth0User)
	}

	// Authenticated endpoints
	protected := api.Group("")
	protected.Use(authMiddleware.Handle())
	{
		auth := protected.Group("/auth")
		{
			auth.POST("/link-identities/confirm", userHandler.ConfirmAccountLink)
		}

		protected.GET("/users/me", userHandler.GetMe)

		tasks := protected.Group("/tasks")
		{
			tasks.POST("", taskHandler.Create)
			tasks.GET("/:id", taskHandler.GetByID)
		}

		ws := protected.Group("/ws")
		{
			ws.GET("/channels/:channelID", chatHandler.HandleWebSocket)
		}
	}
}
