package todo

import (
	"net/http"
	"strconv"

	"todo/common"
	"todo/internal/services"

	"github.com/gin-gonic/gin"
)

// Todo represents a todo item for the handler layer
type Todo struct {
	ID          uint   `json:"id"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// convertServiceTodo converts service Todo to handler Todo
func convertServiceTodo(st *services.Todo) Todo {
	return Todo{
		ID:          st.ID,
		Title:       st.Title,
		Description: st.Description,
		Completed:   st.Completed,
	}
}

// convertToServiceTodo converts handler Todo to service Todo
func convertToServiceTodo(t Todo) *services.Todo {
	return &services.Todo{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Completed:   t.Completed,
	}
}

// GetAll returns all todos
func (h *TodoHandler) GetAll(c *gin.Context) {
	todos, err := h.service.GetAllTodos()
	if err != nil {
		response := common.InternalServerErrorResponse("Failed to retrieve todos")
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Convert service todos to handler todos
	handlerTodos := make([]Todo, len(todos))
	for i, t := range todos {
		handlerTodos[i] = convertServiceTodo(&t)
	}

	response := common.SuccessResponseWithMessage("Todos retrieved successfully", gin.H{
		"todos": handlerTodos,
		"count": len(handlerTodos),
	})
	c.JSON(http.StatusOK, response)
}

// GetByID returns a specific todo by ID
func (h *TodoHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response := common.BadRequestResponse("Invalid ID format")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	todo, err := h.service.GetTodoByID(uint(id))
	if err != nil {
		response := common.NotFoundResponse("Todo not found")
		c.JSON(http.StatusNotFound, response)
		return
	}

	handlerTodo := convertServiceTodo(todo)
	response := common.SuccessResponseWithMessage("Todo retrieved successfully", handlerTodo)
	c.JSON(http.StatusOK, response)
}

// Create creates a new todo
func (h *TodoHandler) Create(c *gin.Context) {
	var todo Todo
	if err := c.ShouldBindJSON(&todo); err != nil {
		response := common.BadRequestResponse("Invalid request body")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	serviceTodo := convertToServiceTodo(todo)
	if err := h.service.CreateTodo(serviceTodo); err != nil {
		response := common.InternalServerErrorResponse("Failed to create todo")
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Update the handler todo with the service todo data
	todo.ID = serviceTodo.ID
	response := common.SuccessResponseWithMessage("Todo created successfully", todo)
	c.JSON(http.StatusCreated, response)
}

// Update updates an existing todo
func (h *TodoHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response := common.BadRequestResponse("Invalid ID format")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var todo Todo
	if err := c.ShouldBindJSON(&todo); err != nil {
		response := common.BadRequestResponse("Invalid request body")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	todo.ID = uint(id)
	serviceTodo := convertToServiceTodo(todo)
	if err := h.service.UpdateTodo(serviceTodo); err != nil {
		response := common.InternalServerErrorResponse("Failed to update todo")
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := common.SuccessResponseWithMessage("Todo updated successfully", todo)
	c.JSON(http.StatusOK, response)
}

// Delete deletes a todo
func (h *TodoHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response := common.BadRequestResponse("Invalid ID format")
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.service.DeleteTodo(uint(id)); err != nil {
		response := common.InternalServerErrorResponse("Failed to delete todo")
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := common.SuccessResponseWithMessage("Todo deleted successfully", nil)
	c.JSON(http.StatusOK, response)
}
