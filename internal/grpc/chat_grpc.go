package grpcserver

import (
	"context"
	"io"

	"todo/internal/services"
	pb "todo/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ChatGRPCHandler implements pb.ChatServiceServer
type ChatGRPCHandler struct {
	pb.UnimplementedChatServiceServer
	chatService *services.ChatService
}

func NewChatGRPCHandler(chatService *services.ChatService) *ChatGRPCHandler {
	return &ChatGRPCHandler{
		chatService: chatService,
	}
}

// SendMessage handles a unary RPC — send one message, get confirmation
func (h *ChatGRPCHandler) SendMessage(ctx context.Context, req *pb.ChatMessageRequest) (*pb.ChatMessageResponse, error) {
	if req.GetRoom() == "" || req.GetSender() == "" || req.GetContent() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "room, sender, and content are required")
	}

	msg, err := h.chatService.SendMessage(ctx, req.GetRoom(), req.GetSender(), req.GetContent())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send message: %v", err)
	}

	return &pb.ChatMessageResponse{
		Id:        msg.ID.String(),
		Room:      msg.Room,
		Sender:    msg.Sender.String(),
		Content:   msg.Content,
		Timestamp: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// ChatStream handles bidirectional streaming — clients send and receive messages in real time
func (h *ChatGRPCHandler) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	// Generate a unique client ID for this connection
	clientID := uuid.New().String()
	room := ""

	// Read the first message to determine the room
	firstMsg, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to receive initial message: %v", err)
	}

	room = firstMsg.GetRoom()
	if room == "" {
		return status.Errorf(codes.InvalidArgument, "first message must specify a room")
	}

	// Join the room and get the broadcast channel
	msgChan := h.chatService.Join(room, clientID)
	defer h.chatService.Leave(room, clientID)

	// Process the first message
	if firstMsg.GetContent() != "" {
		_, err := h.chatService.SendMessage(stream.Context(), room, firstMsg.GetSender(), firstMsg.GetContent())
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send message: %v", err)
		}
	}

	// Goroutine: forward broadcast messages to the client stream
	errChan := make(chan error, 1)
	go func() {
		for msg := range msgChan {
			resp := &pb.ChatMessageResponse{
				Id:        msg.ID.String(),
				Room:      msg.Room,
				Sender:    msg.Sender.String(),
				Content:   msg.Content,
				Timestamp: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
			if err := stream.Send(resp); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Main loop: receive messages from the client
	for {
		select {
		case err := <-errChan:
			return status.Errorf(codes.Internal, "stream send error: %v", err)
		default:
		}

		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "receive error: %v", err)
		}

		if req.GetContent() != "" {
			_, err := h.chatService.SendMessage(stream.Context(), room, req.GetSender(), req.GetContent())
			if err != nil {
				return status.Errorf(codes.Internal, "failed to send message: %v", err)
			}
		}
	}
}
