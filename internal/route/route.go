package route

import (
	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/handler"
	"github.com/Kash4299/todo-chat-app/internal/middleware"

	"github.com/gin-gonic/gin"
)

var allowedRoles = []string{constants.AdminRole, constants.MemberRole}

func SetupRoutes(
	r *gin.Engine,
	userHandler *handler.UserHandler,
	localAuthHandler *handler.LocalAuthHandler,
	taskHandler *handler.TaskHandler,
	chatHandler *handler.ChatHandler,
	workspaceHandler *handler.WorkspaceHandler,
	workspaceInvitationHandler *handler.WorkspaceInvitationHandler,
	authMiddleware *middleware.AuthMiddleware,
	rbacMiddleware *middleware.RBACMiddleware,
) {
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	api := r.Group("/api/v1")
	api.Use(middleware.MaxBodySize(1 << 20))

	// Public webhook endpoints
	webhooks := api.Group("/webhooks")
	{
		webhooks.POST("/auth0/sync", userHandler.SyncAuth0User)
	}

	// Public auth endpoints (email/password)
	auth := api.Group("/auth")
	{
		auth.POST("/register", localAuthHandler.Register)
		auth.POST("/verify-email", localAuthHandler.VerifyEmail)
		auth.POST("/resend-verification", localAuthHandler.ResendVerification)
		auth.POST("/login", localAuthHandler.Login)
		auth.POST("/refresh", localAuthHandler.Refresh)
		auth.POST("/logout", localAuthHandler.Logout)
		auth.POST("/link/confirm", localAuthHandler.ConfirmAccountLink)
	}

	// Authenticated endpoints
	protected := api.Group("")
	protected.Use(authMiddleware.Handle())
	{
		protected.GET("/users/me", userHandler.GetMe)
		protected.PATCH("/users/me", userHandler.UpdateProfile)
		protected.POST("/users/me/password", localAuthHandler.SetPassword)

		tasks := protected.Group("/tasks")
		{
			tasks.POST("", taskHandler.Create)
			tasks.GET("", taskHandler.List)
			tasks.GET("/:id", taskHandler.GetByID)
			tasks.PATCH("/:id", taskHandler.UpdateTask)
			tasks.DELETE("/:id", taskHandler.DeleteTask)
		}

		workspaces := protected.Group("/workspaces")
		{
			workspaces.GET("", workspaceHandler.ListByUser)
			workspaces.GET("/:workspaceID", rbacMiddleware.RequireWorkspaceRole(allowedRoles), workspaceHandler.GetByID)
			workspaces.POST("", workspaceHandler.Create)
			workspaces.DELETE("/:workspaceID", rbacMiddleware.RequireWorkspaceRole([]string{constants.AdminRole}), workspaceHandler.Delete)

			workspaces.POST("/:workspaceID/invitations", rbacMiddleware.RequireWorkspaceRole([]string{constants.AdminRole}), workspaceInvitationHandler.Invite)
			workspaces.POST("/:workspaceID/invitations/resend", rbacMiddleware.RequireWorkspaceRole([]string{constants.AdminRole}), workspaceInvitationHandler.ResendInvitation)
			workspaces.GET("/:workspaceID/invitations", rbacMiddleware.RequireWorkspaceRole([]string{constants.AdminRole}), workspaceInvitationHandler.GetListInvitations)
		}

		protected.POST("/invitations/accept", workspaceInvitationHandler.AcceptInvitation)

		ws := protected.Group("/ws")
		{
			ws.GET("/channels/:channelID", chatHandler.HandleWebSocket)
		}
	}
}
