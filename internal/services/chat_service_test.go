package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"todo/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ─── Mock Repository ─────────────────────────────────────────────

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

func newTestChatService(mockRepo *MockChatRepository) *ChatService {
	hub := &Hub{ChatRepository: mockRepo}
	return NewChatService(hub)
}

// ─── Tests ───────────────────────────────────────────────────────

func TestSendMessage(t *testing.T) {
	senderID := uuid.New().String()

	tests := []struct {
		name      string
		room      string
		sender    string
		content   string
		mockSetup func(m *MockChatRepository)
		wantErr   bool
	}{
		{
			name:    "success - sends message",
			room:    "room-1",
			sender:  senderID,
			content: "Hello, world!",
			mockSetup: func(m *MockChatRepository) {
				m.On("SaveMessage", mock.Anything, mock.AnythingOfType("*model.ChatMessage")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "error - invalid sender UUID",
			room:    "room-1",
			sender:  "not-a-uuid",
			content: "Hello",
			mockSetup: func(m *MockChatRepository) {
				// No mock needed — fails before DB call
			},
			wantErr: true,
		},
		{
			name:    "error - repository failure",
			room:    "room-1",
			sender:  senderID,
			content: "Hello",
			mockSetup: func(m *MockChatRepository) {
				m.On("SaveMessage", mock.Anything, mock.AnythingOfType("*model.ChatMessage")).
					Return(errors.New("db write failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockChatRepository)
			tt.mockSetup(mockRepo)

			svc := newTestChatService(mockRepo)
			msg, err := svc.SendMessage(context.Background(), tt.room, tt.sender, tt.content)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, msg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, msg)
				assert.Equal(t, tt.room, msg.Room)
				assert.Equal(t, tt.content, msg.Content)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestJoinAndLeave(t *testing.T) {
	mockRepo := new(MockChatRepository)
	svc := newTestChatService(mockRepo)

	// Join a room
	ch := svc.Join("room-1", "client-1")
	assert.NotNil(t, ch)

	// Verify channel receives broadcast messages
	mockRepo.On("SaveMessage", mock.Anything, mock.AnythingOfType("*model.ChatMessage")).Return(nil)

	go func() {
		_, _ = svc.SendMessage(context.Background(), "room-1", uuid.New().String(), "test message")
	}()

	select {
	case msg := <-ch:
		assert.Equal(t, "room-1", msg.Room)
		assert.Equal(t, "test message", msg.Content)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast message")
	}

	// Leave the room
	svc.Leave("room-1", "client-1")

	// Channel should be closed
	_, open := <-ch
	assert.False(t, open)
}

func TestGetRoomHistory(t *testing.T) {
	tests := []struct {
		name      string
		room      string
		limit     int
		mockSetup func(m *MockChatRepository)
		wantErr   bool
		wantLen   int
	}{
		{
			name:  "success - returns messages",
			room:  "room-1",
			limit: 50,
			mockSetup: func(m *MockChatRepository) {
				messages := &[]model.ChatMessage{
					{Room: "room-1", Content: "msg1"},
					{Room: "room-1", Content: "msg2"},
				}
				m.On("GetMessagesByRoom", mock.Anything, "room-1", 50).Return(messages, nil)
			},
			wantErr: false,
			wantLen: 2,
		},
		{
			name:  "error - repository failure",
			room:  "room-1",
			limit: 50,
			mockSetup: func(m *MockChatRepository) {
				m.On("GetMessagesByRoom", mock.Anything, "room-1", 50).
					Return(nil, errors.New("db read failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockChatRepository)
			tt.mockSetup(mockRepo)

			svc := newTestChatService(mockRepo)
			result, err := svc.GetRoomHistory(context.Background(), tt.room, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, *result, tt.wantLen)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
