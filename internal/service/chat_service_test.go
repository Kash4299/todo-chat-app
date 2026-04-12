package service

import (
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockChatRepository struct {
	mock.Mock
}

func (m *MockChatRepository) SaveMessage(msg *model.ChatMessage) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockChatRepository) GetMessagesByRoomID(roomID string) ([]model.ChatMessage, error) {
	args := m.Called(roomID)
	if args.Get(0) != nil {
		return args.Get(0).([]model.ChatMessage), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestChatService_RoomManagement(t *testing.T) {
	mockRepo := new(MockChatRepository)
	chatService := NewChatService(mockRepo)
	cs := chatService.(*ChatService)

	roomID := "room1"
	clientChan := make(chan *model.ChatMessage, 10)

	// Register
	chatService.RegisterClient(roomID, clientChan)
	assert.True(t, cs.rooms[roomID][clientChan])

	// Broadcast
	msg := &model.ChatMessage{RoomID: roomID, SenderID: 1, Content: "Hello"}
	chatService.BroadcastToRoom(roomID, msg)
	
	select {
	case receivedMsg := <-clientChan:
		assert.Equal(t, "Hello", receivedMsg.Content)
	default:
		t.Errorf("expected message but got none")
	}

	// Unregister
	chatService.UnregisterClient(roomID, clientChan)
	assert.Nil(t, cs.rooms[roomID])
}
