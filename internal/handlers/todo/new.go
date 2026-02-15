package todo

import "todo/internal/services"

type TodoHandler struct {
	service services.ITodoService
}

func NewTodoHandler(service *services.TodoService) *TodoHandler {
	return &TodoHandler{service: service}
}
