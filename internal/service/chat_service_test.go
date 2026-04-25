package service_test

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/google/uuid"
)

type mockMessageRepo struct {
	saveErr   error
	saveCalls int
	lastMsg   *model.Message
}

func (m *mockMessageRepo) SaveMessage(msg *model.Message) error {
	m.saveCalls++
	m.lastMsg = msg
	return m.saveErr
}

func (m *mockMessageRepo) GetMessagesByTask(taskID uuid.UUID, limit int) ([]model.Message, error) {
	return nil, nil
}

type mockTaskMemberRepo struct {
	isMember bool
	err      error
}

func (m *mockTaskMemberRepo) AddMember(member *model.TaskMember) error {
	return nil
}

func (m *mockTaskMemberRepo) RemoveMember(taskID, userID uuid.UUID) error {
	return nil
}

func (m *mockTaskMemberRepo) IsMember(taskID, userID uuid.UUID) (bool, error) {
	return m.isMember, m.err
}

func (m *mockTaskMemberRepo) GetMembersByTask(taskID uuid.UUID) ([]model.TaskMember, error) {
	return nil, nil
}

func (m *mockTaskMemberRepo) GetTasksByUser(userID uuid.UUID) ([]model.TaskMember, error) {
	return nil, nil
}

func TestChatService_JoinRoomRejectsNonMember(t *testing.T) {
	chatSvc := service.NewChatService(&mockMessageRepo{}, &mockTaskMemberRepo{isMember: false})

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	err := chatSvc.JoinRoom(uuid.New(), uuid.New(), serverConn)
	if err == nil {
		t.Fatal("expected unauthorized error for non-member")
	}
}

func TestChatService_BroadcastToRoomSavesAndWrites(t *testing.T) {
	msgRepo := &mockMessageRepo{}
	chatSvc := service.NewChatService(msgRepo, &mockTaskMemberRepo{isMember: true})

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	taskID := uuid.New()
	userID := uuid.New()
	if err := chatSvc.JoinRoom(taskID, userID, serverConn); err != nil {
		t.Fatalf("join room failed: %v", err)
	}

	readDone := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		signaled := false
		for {
			n, err := clientConn.Read(buf)
			if n > 0 && !signaled {
				close(readDone)
				signaled = true
			}
			if err != nil {
				return
			}
		}
	}()

	msg := &model.Message{
		ID:      uuid.New(),
		TaskID:  taskID,
		UserID:  userID,
		Content: "hello",
	}

	broadcastDone := make(chan struct{})
	go func() {
		defer close(broadcastDone)
		chatSvc.BroadcastToRoom(taskID, msg)
	}()

	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("expected broadcast to reach websocket client")
	}
	select {
	case <-broadcastDone:
	case <-time.After(time.Second):
		t.Fatal("expected broadcast write to complete")
	}

	if msgRepo.saveCalls != 1 {
		t.Fatalf("expected one message save, got %d", msgRepo.saveCalls)
	}
	if msgRepo.lastMsg != msg {
		t.Fatal("expected saved message pointer to match broadcast message")
	}
}

func TestChatService_BroadcastToRoomRemovesBrokenConnection(t *testing.T) {
	msgRepo := &mockMessageRepo{}
	chatSvc := service.NewChatService(msgRepo, &mockTaskMemberRepo{isMember: true})

	taskID := uuid.New()
	userID := uuid.New()

	serverConn, clientConn := net.Pipe()
	if err := chatSvc.JoinRoom(taskID, userID, serverConn); err != nil {
		t.Fatalf("join room failed: %v", err)
	}
	_ = clientConn.Close()

	msg := &model.Message{
		ID:      uuid.New(),
		TaskID:  taskID,
		UserID:  userID,
		Content: "broken",
	}

	chatSvc.BroadcastToRoom(taskID, msg)

	done := make(chan struct{})
	go func() {
		defer close(done)
		chatSvc.BroadcastToRoom(taskID, msg)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected second broadcast to skip previously broken connection")
	}

	_ = serverConn.Close()
}

func TestChatService_BroadcastToRoomIsConcurrentSafe(t *testing.T) {
	chatSvc := service.NewChatService(&mockMessageRepo{}, &mockTaskMemberRepo{isMember: true})

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	taskID := uuid.New()
	userID := uuid.New()
	if err := chatSvc.JoinRoom(taskID, userID, serverConn); err != nil {
		t.Fatalf("join room failed: %v", err)
	}

	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := clientConn.Read(buf); err != nil {
				return
			}
		}
	}()

	msg := &model.Message{
		ID:      uuid.New(),
		TaskID:  taskID,
		UserID:  userID,
		Content: "Test message",
	}

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			chatSvc.BroadcastToRoom(taskID, msg)
		}()
	}
	wg.Wait()
}

func TestChatService_JoinRoomReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("membership lookup failed")
	chatSvc := service.NewChatService(&mockMessageRepo{}, &mockTaskMemberRepo{err: expectedErr})

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	err := chatSvc.JoinRoom(uuid.New(), uuid.New(), serverConn)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}
