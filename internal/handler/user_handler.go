package handler

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	service service.IUserService
	cfg     *config.Config
}

func NewUserHandler(svc service.IUserService, cfg *config.Config) *UserHandler {
	return &UserHandler{service: svc, cfg: cfg}
}

type auth0SyncWebhookRequest struct {
	Auth0ID       string `json:"auth0_id" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	DisplayName   string `json:"display_name"`
	AvatarURL     string `json:"avatar_url"`
	EmailVerified bool   `json:"email_verified"`
}

func (h *UserHandler) SyncAuth0User(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)

	if strings.TrimSpace(h.cfg.WebhookSecret) == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "webhook is not configured"})
		return
	}

	if !h.validateWebhookSecret(c.GetHeader("X-Webhook-Secret")) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook secret"})
		return
	}

	var req auth0SyncWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	user, err := h.service.SyncAuth0User(req.Auth0ID, req.Email, req.DisplayName, req.AvatarURL, req.EmailVerified)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user identity"})
		return
	}

	user, err := h.service.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) validateWebhookSecret(received string) bool {
	secret := strings.TrimSpace(h.cfg.WebhookSecret)
	if received == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(secret), []byte(received)) == 1
}
