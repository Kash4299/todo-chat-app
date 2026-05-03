package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/Kash4299/todo-chat-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TaskHandler struct {
	service service.ITaskService
}

func NewTaskHandler(svc service.ITaskService) *TaskHandler {
	return &TaskHandler{service: svc}
}

type createTaskRequest struct {
	WorkspaceID string `json:"workspace_id" binding:"required"`
	Title       string `json:"title"        binding:"required"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	DueDate     string `json:"due_date"` // RFC3339
}

func (h *TaskHandler) Create(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid workspace_id")
		return
	}

	userIDVal, _ := c.Get(middleware.UserIDContextKey)
	creatorID, ok := userIDVal.(uuid.UUID)
	if !ok || creatorID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}

	task := &model.Task{
		WorkspaceID: workspaceID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
	}

	if req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, req.DueDate)
		if err != nil {
			response.BadRequest(c, response.CodeInvalidInput, "due_date must be RFC3339 format")
			return
		}
		task.DueDate = &t
	}

	if err := h.service.Create(creatorID, task); err != nil {
		switch {
		case errors.Is(err, constants.ErrForbidden):
			response.Forbidden(c)
		case errors.Is(err, constants.ErrTaskInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid task input")
		default:
			response.InternalError(c)
		}
		return
	}

	response.Created(c, task)
}

func (h *TaskHandler) GetByID(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}
	requesterID, ok := userIDVal.(uuid.UUID)
	if !ok || requesterID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "invalid user identity")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid task id")
		return
	}

	task, err := h.service.GetByID(requesterID, id)
	if err != nil {
		switch {
		case errors.Is(err, constants.ErrTaskInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid task input")
		case errors.Is(err, constants.ErrForbidden):
			response.Forbidden(c)
		case errors.Is(err, constants.ErrTaskNotFound):
			response.NotFound(c, response.CodeNotFound, "task not found")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, task)
}

func (h *TaskHandler) List(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}
	requesterID, ok := userIDVal.(uuid.UUID)
	if !ok || requesterID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "invalid user identity")
		return
	}

	workspaceID, err := uuid.Parse(c.Query("workspace_id"))
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "workspace_id is required and must be a valid UUID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	tasks, total, err := h.service.ListByWorkspace(requesterID, workspaceID, page, pageSize)
	if err != nil {
		switch {
		case errors.Is(err, constants.ErrForbidden):
			response.Forbidden(c)
		case errors.Is(err, constants.ErrTaskInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid input")
		default:
			response.InternalError(c)
		}
		return
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response.List(c, tasks, &response.PaginationMeta{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	DueDate     *string `json:"due_date"`   // nil=skip, ""=clear, "RFC3339"=set
	AssigneeID  *string `json:"assignee_id"` // nil=skip, ""=unassign
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}
	requesterID, ok := userIDVal.(uuid.UUID)
	if !ok || requesterID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "invalid user identity")
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid task id")
		return
	}

	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid request body")
		return
	}

	task, err := h.service.UpdateTask(requesterID, taskID, service.TaskUpdateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		switch {
		case errors.Is(err, constants.ErrForbidden):
			response.Forbidden(c)
		case errors.Is(err, constants.ErrTaskNotFound):
			response.NotFound(c, response.CodeNotFound, "task not found")
		case errors.Is(err, constants.ErrTaskInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid task input")
		default:
			response.InternalError(c)
		}
		return
	}

	response.OK(c, task)
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}
	requesterID, ok := userIDVal.(uuid.UUID)
	if !ok || requesterID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "invalid user identity")
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid task id")
		return
	}

	if err := h.service.DeleteTask(requesterID, taskID); err != nil {
		switch {
		case errors.Is(err, constants.ErrForbidden):
			response.Forbidden(c)
		case errors.Is(err, constants.ErrTaskNotFound):
			response.NotFound(c, response.CodeNotFound, "task not found")
		case errors.Is(err, constants.ErrTaskInvalidInput):
			response.BadRequest(c, response.CodeInvalidInput, "invalid task input")
		default:
			response.InternalError(c)
		}
		return
	}

	response.NoContent(c)
}
