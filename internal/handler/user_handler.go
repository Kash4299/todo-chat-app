package handler

import (
	"crypto/subtle"
	"strings"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/Kash4299/todo-chat-app/pkg/response"
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
	Auth0ID       string `json:"auth0_id"       binding:"required"`
	Email         string `json:"email"          binding:"required,email"`
	DisplayName   string `json:"display_name"`
	AvatarURL     string `json:"avatar_url"`
	EmailVerified bool   `json:"email_verified"`
}

func (h *UserHandler) SyncAuth0User(c *gin.Context) {
	if strings.TrimSpace(h.cfg.WebhookSecret) == "" {
		response.InternalError(c)
		return
	}

	if !h.validateWebhookSecret(c.GetHeader("X-Webhook-Secret")) {
		response.Unauthorized(c, response.CodeUnauthorized, "invalid webhook secret")
		return
	}

	var req auth0SyncWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	user, err := h.service.SyncAuth0User(req.Auth0ID, req.Email, req.DisplayName, req.AvatarURL, req.EmailVerified)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, user)
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "invalid user identity")
		return
	}

	user, err := h.service.GetByID(userID)
	if err != nil {
		response.NotFound(c, response.CodeNotFound, "user not found")
		return
	}

	response.OK(c, user)
}

func (h *UserHandler) validateWebhookSecret(received string) bool {
	secret := strings.TrimSpace(h.cfg.WebhookSecret)
	if received == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(secret), []byte(received)) == 1
}
