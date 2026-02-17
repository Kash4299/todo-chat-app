package grpcserver

import (
	"context"
	"errors"
	"testing"

	"todo/internal/model"
	"todo/internal/services"
	pb "todo/proto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ─── Mock Chat Repository (for building ChatService in tests) ─────

type MockChatRepository struct {
	mock.Mock
}

func (m *MockChatRepository) SaveMessage(ctx context.Context, msg *model.ChatMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockChatRepository) GetMessagesByRoom(ctx context.Context, room string, limit int) (*[]model.ChatMessage, error) {
	args := m.Called(ctx, room, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]model.ChatMessage), args.Error(1)
}

// ─── Helper ──────────────────────────────────────────────────────

func newTestChatGRPCHandler(mockRepo *MockChatRepository) *ChatGRPCHandler {
	hub := &services.Hub{ChatRepository: mockRepo}
	chatService := services.NewChatService(hub)
	return &ChatGRPCHandler{
		chatService: chatService,
	}
}

// ─── Tests: SendMessage (unary RPC) ──────────────────────────────

func TestGRPC_SendMessage(t *testing.T) {
	senderID := uuid.New().String()

	tests := []struct {
		name      string
		req       *pb.ChatMessageRequest
		mockSetup func(m *MockChatRepository)
		wantErr   bool
		wantCode  codes.Code
	}{
		{
			name: "success - sends message",
			req: &pb.ChatMessageRequest{
				Room:    "room-1",
				Sender:  senderID,
				Content: "Hello gRPC!",
			},
			mockSetup: func(m *MockChatRepository) {
				m.On("SaveMessage", mock.Anything, mock.AnythingOfType("*model.ChatMessage")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - missing fields",
			req: &pb.ChatMessageRequest{
				Room:    "",
				Sender:  "",
				Content: "",
			},
			mockSetup: func(m *MockChatRepository) {},
			wantErr:   true,
			wantCode:  codes.InvalidArgument,
		},
		{
			name: "error - service failure",
			req: &pb.ChatMessageRequest{
				Room:    "room-1",
				Sender:  senderID,
				Content: "Hello",
			},
			mockSetup: func(m *MockChatRepository) {
				m.On("SaveMessage", mock.Anything, mock.AnythingOfType("*model.ChatMessage")).
					Return(errors.New("db down"))
			},
			wantErr:  true,
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockChatRepository)
			tt.mockSetup(mockRepo)

			handler := newTestChatGRPCHandler(mockRepo)
			resp, err := handler.SendMessage(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, "room-1", resp.Room)
				assert.Equal(t, "Hello gRPC!", resp.Content)
				assert.NotEmpty(t, resp.Id)
				assert.NotEmpty(t, resp.Timestamp)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
