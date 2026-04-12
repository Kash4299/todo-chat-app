package service_test

import (
	"net"
	"sync"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/google/uuid"
)

// Mock repositories implementation
type mockMessageRepo struct{}

func (m *mockMessageRepo) SaveMessage(msg *model.Message) error { return nil }
func (m *mockMessageRepo) GetMessagesByTask(taskID uuid.UUID, limit int) ([]model.Message, error) {
	return nil, nil
}

type mockTaskMemberRepo struct {
	isMember bool
}

func (m *mockTaskMemberRepo) AddMember(member *model.TaskMember) error                    { return nil }
func (m *mockTaskMemberRepo) RemoveMember(taskID, userID uuid.UUID) error                 { return nil }
func (m *mockTaskMemberRepo) IsMember(taskID, userID uuid.UUID) (bool, error)             { return m.isMember, nil }
func (m *mockTaskMemberRepo) GetMembersByTask(taskID uuid.UUID) ([]model.TaskMember, error) { return nil, nil }
func (m *mockTaskMemberRepo) GetTasksByUser(userID uuid.UUID) ([]model.TaskMember, error)   { return nil, nil }

func TestChatService_JoinAndBroadcast(t *testing.T) {
	msgRepo := &mockMessageRepo{}
	tmRepo := &mockTaskMemberRepo{isMember: true}

	chatSvc := service.NewChatService(msgRepo, tmRepo)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	taskID := uuid.New()
	userID := uuid.New()

	// This is intentionally passing net.Conn. If the interface expects *websocket.Conn, it will fail to compile.
	err := chatSvc.JoinRoom(taskID, userID, serverConn)
	if err != nil {
		t.Fatalf("Expected no error joining room, got: %v", err)
	}

	// Set up concurrent broadcast
	var wg sync.WaitGroup
	wg.Add(2)

	msg := &model.Message{
		ID:       uuid.New(),
		TaskID:   taskID,
		UserID:   userID,
		Content:  "Test message",
	}

	go func() {
		defer wg.Done()
		chatSvc.BroadcastToRoom(taskID, msg)
	}()
	go func() {
		defer wg.Done()
		chatSvc.BroadcastToRoom(taskID, msg)
	}()

	// Read from client connection to unblock write (if using synchronous pipe)
	go func() {
		buf := make([]byte, 1024)
		for {
			_, err := clientConn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	wg.Wait()
}
