package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
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
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)

	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
		return
	}

	userIDVal, _ := c.Get(middleware.UserIDContextKey)
	creatorID, ok := userIDVal.(uuid.UUID)
	if !ok || creatorID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "due_date must be RFC3339 format"})
			return
		}
		task.DueDate = &t
	}

	if err := h.service.Create(creatorID, task); err != nil {
		if errors.Is(err, service.ErrTaskForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if errors.Is(err, service.ErrTaskInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task input"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *TaskHandler) GetByID(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	requesterID, ok := userIDVal.(uuid.UUID)
	if !ok || requesterID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user identity"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	task, err := h.service.GetByID(requesterID, id)
	if err != nil {
		if errors.Is(err, service.ErrTaskForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if errors.Is(err, service.ErrTaskInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}
		if errors.Is(err, service.ErrTaskNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}
