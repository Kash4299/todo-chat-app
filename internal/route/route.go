package route

import (
	"github.com/Kash4299/todo-chat-app/internal/handler"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	r *gin.Engine,
	userHandler *handler.UserHandler,
	localAuthHandler *handler.LocalAuthHandler,
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

	// Public auth endpoints (email/password)
	auth := api.Group("/auth")
	{
		auth.POST("/register", localAuthHandler.Register)
		auth.POST("/login", localAuthHandler.Login)
		auth.POST("/refresh", localAuthHandler.Refresh)
		auth.POST("/logout", localAuthHandler.Logout)
	}

	// Authenticated endpoints
	protected := api.Group("")
	protected.Use(authMiddleware.Handle())
	{
		protected.GET("/users/me", userHandler.GetMe)
		protected.POST("/users/me/password", localAuthHandler.SetPassword)

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
