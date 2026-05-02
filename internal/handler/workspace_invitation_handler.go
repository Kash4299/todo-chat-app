package handler

import (
	"errors"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/Kash4299/todo-chat-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkspaceInvitationHandler struct {
	service service.IWorkspaceInvitationService
}

func NewWorkspaceInvitationHandler(svc service.IWorkspaceInvitationService) *WorkspaceInvitationHandler {
	return &WorkspaceInvitationHandler{service: svc}
}

type createWorkspaceInvitationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *WorkspaceInvitationHandler) Invite(c *gin.Context) {
	var req createWorkspaceInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	userIDVal, _ := c.Get(middleware.UserIDContextKey)
	inviterID, ok := userIDVal.(uuid.UUID)
	if !ok || inviterID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}

	workspaceIDStr := c.Param("workspaceID")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid workspace ID")
		return
	}

	err = h.service.Invite(workspaceID, inviterID, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, constants.ErrWorkspaceInvitationInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid invitation input")
		case errors.Is(err, constants.ErrWorkspaceInvitationNotAdmin):
			response.Forbidden(c)
		case errors.Is(err, constants.ErrWorkspaceInvitationIsMember):
			response.BadRequest(c, response.CodeInvalidInput, "user is already a member of the workspace")
		case errors.Is(err, constants.ErrWorkspaceInvitationStillValid):
			response.BadRequest(c, response.CodeInvalidInput, "an invitation for this email already exists and is still valid")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, gin.H{"message": "invitation sent"})
}

func (h *WorkspaceInvitationHandler) ResendInvitation(c *gin.Context) {
	var req createWorkspaceInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	userIDVal, _ := c.Get(middleware.UserIDContextKey)
	inviterID, ok := userIDVal.(uuid.UUID)
	if !ok || inviterID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}

	workspaceIDStr := c.Param("workspaceID")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid workspace ID")
		return
	}

	err = h.service.ResendInvitation(workspaceID, inviterID, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, constants.ErrWorkspaceInvitationInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid invitation input")
		case errors.Is(err, constants.ErrWorkspaceInvitationNotAdmin):
			response.Forbidden(c)
		case errors.Is(err, constants.ErrWorkspaceInvitationIsMember):
			response.BadRequest(c, response.CodeInvalidInput, "user is already a member of the workspace")
		case errors.Is(err, constants.ErrWorkspaceInvitationStillValid):
			response.BadRequest(c, response.CodeInvalidInput, "an invitation for this email already exists and is still valid")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, gin.H{"message": "invitation resent"})
}

func (h *WorkspaceInvitationHandler) GetListInvitations(c *gin.Context) {
	workspaceIDStr := c.Param("workspaceID")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid workspace ID")
		return
	}

	invitations, err := h.service.GetListInvitations(workspaceID)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, invitations)
}

type acceptInvitationRequest struct {
	Token string `json:"token" binding:"required"`
}

func (h *WorkspaceInvitationHandler) AcceptInvitation(c *gin.Context) {
	userIDVal, _ := c.Get(middleware.UserIDContextKey)
	actorID, ok := userIDVal.(uuid.UUID)
	if !ok || actorID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}

	var req acceptInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "token is required")
		return
	}

	workspaceID, err := h.service.AcceptInvitation(actorID, req.Token)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.NotFound(c, response.CodeNotFound, "invitation not found")
		case errors.Is(err, constants.ErrWorkspaceInvitationInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid invitation")
		case errors.Is(err, constants.ErrWorkspaceInvitationIsMember):
			response.BadRequest(c, response.CodeInvalidInput, "you are already a member of the workspace")
		case errors.Is(err, constants.ErrWorkspaceInvitationIsExpired):
			response.BadRequest(c, response.CodeInvalidInput, "invitation has expired")
		case errors.Is(err, constants.ErrWorkspaceInvitationEmailMismatch):
			response.Forbidden(c)
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, gin.H{"workspace_id": workspaceID})
}
