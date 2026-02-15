package grpcserver

import (
	"context"

	"todo/internal/dtos/request"
	"todo/internal/services"
	pb "todo/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserGRPCHandler implements todopb.UserServiceServer
type UserGRPCHandler struct {
	pb.UnimplementedUserServiceServer
	userService services.IUserService
}

func NewUserGRPCHandler(userService *services.UserService) *UserGRPCHandler {
	return &UserGRPCHandler{
		userService: userService,
	}
}

func (h *UserGRPCHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	createReq := &request.CreateUserRequest{
		Username: req.GetUsername(),
		Password: req.GetPassword(),
		Email:    req.GetEmail(),
	}

	if err := h.userService.CreateUser(ctx, createReq); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return &pb.CreateUserResponse{Message: "User created successfully"}, nil
}

func (h *UserGRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	loginReq := &request.UserLoginRequest{
		Username: req.GetUsername(),
		Password: req.GetPassword(),
	}

	token, err := h.userService.UserLogin(ctx, loginReq)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "login failed: %v", err)
	}

	return &pb.LoginResponse{Token: token}, nil
}
