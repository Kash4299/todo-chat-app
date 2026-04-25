package handler

import (
	"crypto/subtle"
	"errors"
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

type confirmAccountLinkRequest struct {
	Consent bool `json:"consent"`
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

func (h *UserHandler) ConfirmAccountLink(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)

	var req confirmAccountLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	auth0ID, _ := c.Get(middleware.Auth0IDContextKey)
	email, _ := c.Get(middleware.EmailContextKey)
	displayName, _ := c.Get(middleware.DisplayNameContextKey)
	avatarURL, _ := c.Get(middleware.AvatarURLContextKey)
	emailVerified, _ := c.Get(middleware.EmailVerifiedContextKey)
	stepUpVerified, _ := c.Get(middleware.StepUpVerifiedContextKey)

	auth0IDStr, _ := auth0ID.(string)
	emailStr, _ := email.(string)
	displayNameStr, _ := displayName.(string)
	avatarURLStr, _ := avatarURL.(string)
	emailVerifiedBool, _ := emailVerified.(bool)
	stepUpVerifiedBool, _ := stepUpVerified.(bool)

	user, err := h.service.ConfirmAccountLink(
		auth0IDStr, emailStr, displayNameStr, avatarURLStr,
		emailVerifiedBool, req.Consent, stepUpVerifiedBool,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserLinkingConsentRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "consent is required"})
		case errors.Is(err, service.ErrUserStepUpRequired):
			c.JSON(http.StatusForbidden, gin.H{"error": "step-up authentication required"})
		case errors.Is(err, service.ErrUserEmailNotVerified):
			c.JSON(http.StatusForbidden, gin.H{"error": "email is not verified"})
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found for linking"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
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
