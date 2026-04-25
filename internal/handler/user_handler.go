package handler

import (
	"net/http"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service service.IUserService
	secret  string
}

func NewUserHandler(svc service.IUserService, cfg *config.Config) *UserHandler {
	return &UserHandler{service: svc, secret: cfg.WebhookSecret}
}

type SyncAuth0Request struct {
	Auth0ID     string `json:"auth0_id" binding:"required"`
	Email       string `json:"email" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	AvatarURL   string `json:"avatar_url"`
}

func (h *UserHandler) SyncAuth0Webhook(c *gin.Context) {
	if h.secret != "" && c.GetHeader("X-Webhook-Secret") != h.secret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)

	var req SyncAuth0Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.SyncAuth0User(req.Auth0ID, req.Email, req.DisplayName, req.AvatarURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, user)
}
