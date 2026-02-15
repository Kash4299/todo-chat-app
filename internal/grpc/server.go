package grpcserver

import (
	"fmt"
	"net"

	"todo/internal/config"
	pb "todo/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC server
type Server struct {
	config      *config.Config
	logger      *zap.Logger
	grpcServer  *grpc.Server
	todoHandler *TodoGRPCHandler
	userHandler *UserGRPCHandler
}

func NewServer(
	config *config.Config,
	logger *zap.Logger,
	todoHandler *TodoGRPCHandler,
	userHandler *UserGRPCHandler,
) *Server {
	grpcServer := grpc.NewServer()

	// Register service implementations
	pb.RegisterTodoServiceServer(grpcServer, todoHandler)
	pb.RegisterUserServiceServer(grpcServer, userHandler)

	// Register reflection service for tools like grpcurl
	reflection.Register(grpcServer)

	return &Server{
		config:      config,
		logger:      logger,
		grpcServer:  grpcServer,
		todoHandler: todoHandler,
		userHandler: userHandler,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.config.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.logger.Info("Starting gRPC server",
		zap.String("port", s.config.GRPC.Port),
	)

	return s.grpcServer.Serve(lis)
}

// GracefulStop gracefully stops the gRPC server
func (s *Server) GracefulStop() {
	s.logger.Info("Stopping gRPC server...")
	s.grpcServer.GracefulStop()
	s.logger.Info("gRPC server stopped")
}

// IsEnabled returns whether gRPC is enabled in config
func (s *Server) IsEnabled() bool {
	return s.config.GRPC.Enabled
}
