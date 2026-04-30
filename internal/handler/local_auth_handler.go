package handler

import (
	"errors"
	"net/http"

	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/Kash4299/todo-chat-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LocalAuthHandler struct {
	service service.ILocalAuthService
}

func NewLocalAuthHandler(svc service.ILocalAuthService) *LocalAuthHandler {
	return &LocalAuthHandler{service: svc}
}

type registerRequest struct {
	Email       string `json:"email"        binding:"required,email"`
	Password    string `json:"password"     binding:"required"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type setPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

type confirmLinkRequest struct {
	PendingToken string `json:"pending_token" binding:"required"`
	Password     string `json:"password"      binding:"required"`
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type resendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *LocalAuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	user, err := h.service.Register(req.Email, req.Password, req.DisplayName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			response.Conflict(c, response.CodeEmailTaken, "email already registered")
		case errors.Is(err, service.ErrPasswordTooShort):
			response.BadRequest(c, response.CodePasswordTooShort, "password must be at least 8 characters")
		default:
			response.InternalError(c)
		}
		return
	}

	response.Created(c, gin.H{
		"user":    user,
		"message": "verification email sent; please check your inbox",
	})
}

func (h *LocalAuthHandler) VerifyEmail(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)

	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	user, pair, err := h.service.VerifyEmail(req.Token)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidVerificationToken):
			response.BadRequest(c, response.CodeTokenInvalid, "invalid or expired verification token")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, gin.H{"user": user, "tokens": pair})
}

func (h *LocalAuthHandler) ResendVerification(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)

	var req resendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	if err := h.service.ResendVerification(req.Email); err != nil {
		switch {
		case errors.Is(err, service.ErrVerificationEmailRateLimited):
			response.OK(c, gin.H{"message": "if the account exists and is unverified, a verification email has been sent"})
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, gin.H{"message": "if the account exists and is unverified, a verification email has been sent"})
}

func (h *LocalAuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	user, pair, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			response.Unauthorized(c, response.CodeInvalidCredentials, "invalid email or password")
		case errors.Is(err, service.ErrNoPasswordSet):
			response.Unauthorized(c, response.CodeInvalidCredentials, "this account uses Google login; no password is set")
		case errors.Is(err, service.ErrEmailNotVerified):
			response.ForbiddenCode(c, response.CodeEmailNotVerified, "email not verified; please check your inbox")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, gin.H{"user": user, "tokens": pair})
}

func (h *LocalAuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	pair, err := h.service.Refresh(req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, response.CodeTokenInvalid, "invalid or expired refresh token")
		return
	}

	response.OK(c, gin.H{"tokens": pair})
}

func (h *LocalAuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	_ = h.service.Logout(req.RefreshToken)
	response.NoContent(c)
}

func (h *LocalAuthHandler) ConfirmAccountLink(c *gin.Context) {
	var req confirmLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	user, pair, err := h.service.ConfirmAccountLink(req.PendingToken, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidPendingToken):
			response.BadRequest(c, response.CodeTokenInvalid, "invalid or expired link token")
		case errors.Is(err, service.ErrInvalidCredentials):
			response.Unauthorized(c, response.CodeInvalidCredentials, "invalid password")
		case errors.Is(err, service.ErrNoPasswordSet):
			response.Unauthorized(c, response.CodeInvalidCredentials, "this account uses Google login only; no local password is set")
		case errors.Is(err, service.ErrLinkConflict):
			response.Conflict(c, response.CodeConflict, "google identity is already linked to a different account")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, gin.H{"user": user, "tokens": pair})
}

func (h *LocalAuthHandler) SetPassword(c *gin.Context) {
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

	var req setPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	if err := h.service.SetPassword(userID, req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrPasswordAlreadySet):
			response.Conflict(c, response.CodePasswordAlreadySet, "password already set; use change-password flow")
		case errors.Is(err, service.ErrPasswordTooShort):
			response.BadRequest(c, response.CodePasswordTooShort, "password must be at least 8 characters")
		default:
			response.InternalError(c)
		}
		return
	}

	response.NoContent(c)
}
