package handler

import (
	"errors"
	"math"
	"strconv"

	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/Kash4299/todo-chat-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WorkspaceHandler struct {
	service service.IWorkspaceService
}

func NewWorkspaceHandler(svc service.IWorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{service: svc}
}

type createWorkspaceRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *WorkspaceHandler) Create(c *gin.Context) {
	var req createWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	userIDVal, _ := c.Get(middleware.UserIDContextKey)
	creatorID, ok := userIDVal.(uuid.UUID)
	if !ok || creatorID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}

	ws, err := h.service.Create(creatorID, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWorkspaceInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid workspace name")
		default:
			response.InternalError(c)
		}
		return
	}

	response.Created(c, ws)
}

func (h *WorkspaceHandler) GetByID(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}
	actorID, ok := userIDVal.(uuid.UUID)
	if !ok || actorID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "invalid user identity")
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid workspace id")
		return
	}

	ws, err := h.service.GetByID(actorID, workspaceID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWorkspaceInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid input")
		case errors.Is(err, service.ErrWorkspaceNotFound):
			response.NotFound(c, response.CodeWorkspaceNotFound, "workspace not found")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, ws)
}

func (h *WorkspaceHandler) ListByUser(c *gin.Context) {
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

	page := 1
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	pageSize := 20
	if ps := c.Query("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 && n <= 100 {
			pageSize = n
		}
	}

	workspaces, total, err := h.service.ListByUser(userID, page, pageSize)
	if err != nil {
		response.InternalError(c)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	response.List(c, workspaces, &response.PaginationMeta{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

func (h *WorkspaceHandler) Delete(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}
	actorID, ok := userIDVal.(uuid.UUID)
	if !ok || actorID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "invalid user identity")
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid workspace id")
		return
	}

	if err := h.service.Delete(actorID, workspaceID); err != nil {
		switch {
		case errors.Is(err, service.ErrWorkspaceInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid input")
		case errors.Is(err, service.ErrWorkspaceNotFound):
			response.NotFound(c, response.CodeWorkspaceNotFound, "workspace not found")
		case errors.Is(err, service.ErrWorkspaceForbidden):
			response.Forbidden(c)
		default:
			response.InternalError(c)
		}
		return
	}

	response.NoContent(c)
}
