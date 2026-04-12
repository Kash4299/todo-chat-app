package handler

import (
	"net/http"

	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service service.IUserService
}

func NewUserHandler(service service.IUserService) *UserHandler {
	return &UserHandler{service: service}
}

type SyncAuth0Request struct {
	Auth0ID     string `json:"auth0_id" binding:"required"`
	Email       string `json:"email" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	AvatarURL   string `json:"avatar_url"`
}

func (h *UserHandler) SyncAuth0Webhook(c *gin.Context) {
	var req SyncAuth0Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.SyncAuth0User(req.Auth0ID, req.Email, req.DisplayName, req.AvatarURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}
