package todo

import (
	"net/http"

	"todo/common"
	"todo/internal/dtos/request"

	"github.com/gin-gonic/gin"
)

func (h *TodoHandler) GetAll(c *gin.Context) {
	ctx := c.Request.Context()
	todos, err := h.service.GetAllTodos(ctx)
	if err != nil {
		response := common.InternalServerErrorResponse("Failed to retrieve todos")
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := common.SuccessResponseWithMessage("Todos retrieved successfully", todos)
	c.JSON(http.StatusOK, response)
}

func (h *TodoHandler) Create(c *gin.Context) {
	var (
		ctx = c.Request.Context()
		req = new(request.CreateTodoRequest)
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		response := common.BadRequestResponse("Invalid request body")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	err := h.service.CreateTodo(ctx, req)
	if err != nil {
		response := common.InternalServerErrorResponse("Failed to create todo")
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := common.SuccessResponseWithMessage("Todo created successfully", nil)
	c.JSON(http.StatusCreated, response)
}
