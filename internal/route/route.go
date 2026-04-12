package route

import (
	"github.com/Kash4299/todo-chat-app/internal/handler"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, userHandler *handler.UserHandler, taskHandler *handler.TaskHandler, chatHandler *handler.ChatHandler) {
	// Root ping
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	api := r.Group("/api/v1")
	{
		// Auth0 sync webhooks (public or secret-protected)
		api.POST("/webhooks/auth0/sync", userHandler.SyncAuth0Webhook)

		tasks := api.Group("/tasks")
		{
			tasks.POST("", taskHandler.Create)
			tasks.GET("/:id", taskHandler.GetByID)
			// Put/Delete omitted for brevity, logic handlers created earlier in TaskService
		}

		// WebSockets map tightly onto actual tasks
		ws := api.Group("/ws")
		{
			// Example URL query params: ws://foo/api/v1/ws/chat/TASK_UUID?userID=USER_UUID
			ws.GET("/chat/:taskID", chatHandler.HandleWebSocket)
		}
	}
}
