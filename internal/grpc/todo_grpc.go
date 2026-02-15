package grpcserver

import (
	"context"

	"todo/internal/dtos/request"
	"todo/internal/model"
	"todo/internal/services"
	pb "todo/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TodoGRPCHandler implements todopb.TodoServiceServer
type TodoGRPCHandler struct {
	pb.UnimplementedTodoServiceServer
	todoService services.ITodoService
}

func NewTodoGRPCHandler(todoService *services.TodoService) *TodoGRPCHandler {
	return &TodoGRPCHandler{
		todoService: todoService,
	}
}

func (h *TodoGRPCHandler) GetAllTodos(ctx context.Context, req *pb.GetAllTodosRequest) (*pb.GetAllTodosResponse, error) {
	todos, err := h.todoService.GetAllTodos(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get todos: %v", err)
	}

	var items []*pb.TodoItem
	for _, t := range *todos {
		items = append(items, &pb.TodoItem{
			Id:          t.ID.String(),
			Title:       t.Title,
			Status:      string(t.Status),
			Priority:    int32(t.Priority),
			Description: t.Description,
			CreatedBy:   t.CreatedBy.String(),
			Assigned:    t.Assigned.String(),
			CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return &pb.GetAllTodosResponse{Todos: items}, nil
}

func (h *TodoGRPCHandler) CreateTodo(ctx context.Context, req *pb.CreateTodoRequest) (*pb.CreateTodoResponse, error) {
	assignedUUID, err := uuid.Parse(req.GetAssigned())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assigned UUID: %v", err)
	}

	createReq := &request.CreateTodoRequest{
		Title:       req.GetTitle(),
		Status:      model.TodoStatus(req.GetStatus()),
		Priority:    model.TodoPriority(req.GetPriority()),
		Description: req.GetDescription(),
		Assigned:    assignedUUID,
	}

	if err := h.todoService.CreateTodo(ctx, createReq); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create todo: %v", err)
	}

	return &pb.CreateTodoResponse{Message: "Todo created successfully"}, nil
}
