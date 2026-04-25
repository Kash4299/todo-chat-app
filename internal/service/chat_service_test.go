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

// ── Mocks ────────────────────────────────────────────────────────────────────

type mockMessageRepo struct {
	mu        sync.Mutex
	saveErr   error
	saveCalls int
	lastMsg   *model.Message
}

func (m *mockMessageRepo) Save(msg *model.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveCalls++
	m.lastMsg = msg
	return m.saveErr
}

func (m *mockMessageRepo) FindByChannelID(channelID uuid.UUID, limit int, beforeID *uuid.UUID) ([]model.Message, error) {
	return nil, nil
}

func (m *mockMessageRepo) Search(channelID uuid.UUID, query string, limit int) ([]model.Message, error) {
	return nil, nil
}

type mockChannelMemberRepo struct {
	isMember bool
	err      error
}

func (m *mockChannelMemberRepo) Add(member *model.ChannelMember) error    { return nil }
func (m *mockChannelMemberRepo) Remove(channelID, userID uuid.UUID) error { return nil }
func (m *mockChannelMemberRepo) IsMember(channelID, userID uuid.UUID) (bool, error) {
	return m.isMember, m.err
}
func (m *mockChannelMemberRepo) FindByChannel(channelID uuid.UUID) ([]model.ChannelMember, error) {
	return nil, nil
}
func (m *mockChannelMemberRepo) FindByUser(userID uuid.UUID) ([]model.ChannelMember, error) {
	return nil, nil
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestChatService_JoinRoomRejectsNonMember(t *testing.T) {
	svc := service.NewChatService(&mockMessageRepo{}, &mockChannelMemberRepo{isMember: false})

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	err := svc.JoinRoom(uuid.New(), uuid.New(), srv)
	if err == nil {
		t.Fatal("expected error for non-member, got nil")
	}
}

func TestChatService_JoinRoomReturnsRepositoryError(t *testing.T) {
	expected := errors.New("db error")
	svc := service.NewChatService(&mockMessageRepo{}, &mockChannelMemberRepo{err: expected})

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	err := svc.JoinRoom(uuid.New(), uuid.New(), srv)
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestChatService_BroadcastToRoomSavesAndWrites(t *testing.T) {
	msgRepo := &mockMessageRepo{}
	svc := service.NewChatService(msgRepo, &mockChannelMemberRepo{isMember: true})

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	channelID := uuid.New()
	userID := uuid.New()

	if err := svc.JoinRoom(channelID, userID, srv); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}

	readDone := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		signaled := false
		for {
			n, err := cli.Read(buf)
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
		ID:        uuid.New(),
		ChannelID: channelID,
		UserID:    userID,
		Content:   "hello",
	}

	broadcastDone := make(chan struct{})
	go func() {
		defer close(broadcastDone)
		svc.BroadcastToRoom(channelID, msg)
	}()

	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("broadcast did not reach client within timeout")
	}
	select {
	case <-broadcastDone:
	case <-time.After(time.Second):
		t.Fatal("BroadcastToRoom did not complete within timeout")
	}

	if msgRepo.saveCalls != 1 {
		t.Fatalf("expected 1 save call, got %d", msgRepo.saveCalls)
	}
	if msgRepo.lastMsg != msg {
		t.Fatal("saved message pointer does not match broadcast message")
	}
}

func TestChatService_BroadcastToRoomRemovesBrokenConnection(t *testing.T) {
	svc := service.NewChatService(&mockMessageRepo{}, &mockChannelMemberRepo{isMember: true})

	channelID := uuid.New()
	srv, cli := net.Pipe()

	if err := svc.JoinRoom(channelID, uuid.New(), srv); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}
	cli.Close()

	msg := &model.Message{ChannelID: channelID, Content: "broken"}
	svc.BroadcastToRoom(channelID, msg)

	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.BroadcastToRoom(channelID, msg)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("second broadcast should complete quickly after connection removed")
	}
}

func TestChatService_BroadcastToRoomIsConcurrentSafe(t *testing.T) {
	svc := service.NewChatService(&mockMessageRepo{}, &mockChannelMemberRepo{isMember: true})

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	channelID := uuid.New()
	if err := svc.JoinRoom(channelID, uuid.New(), srv); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}

	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := cli.Read(buf); err != nil {
				return
			}
		}
	}()

	msg := &model.Message{ChannelID: channelID, Content: "concurrent"}
	var wg sync.WaitGroup
	wg.Add(3)
	for range 3 {
		go func() {
			defer wg.Done()
			svc.BroadcastToRoom(channelID, msg)
		}()
	}
	wg.Wait()
}

func TestChatService_LeaveRoomRemovesLastConnection(t *testing.T) {
	svc := service.NewChatService(&mockMessageRepo{}, &mockChannelMemberRepo{isMember: true})
	channelID := uuid.New()
	srv, cli := net.Pipe()
	defer cli.Close()

	if err := svc.JoinRoom(channelID, uuid.New(), srv); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}
	svc.LeaveRoom(channelID, srv)

	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.BroadcastToRoom(channelID, &model.Message{ChannelID: channelID, Content: "x"})
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("broadcast should return quickly for empty room")
	}
}

func TestChatService_BroadcastToRoomNoRoom(t *testing.T) {
	svc := service.NewChatService(&mockMessageRepo{}, &mockChannelMemberRepo{isMember: true})
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.BroadcastToRoom(uuid.New(), &model.Message{Content: "x"})
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("broadcast should return quickly when room does not exist")
	}
}

func TestChatService_BroadcastToRoomSaveErrorStillBroadcasts(t *testing.T) {
	msgRepo := &mockMessageRepo{saveErr: errors.New("save failed")}
	svc := service.NewChatService(msgRepo, &mockChannelMemberRepo{isMember: true})

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	channelID := uuid.New()
	if err := svc.JoinRoom(channelID, uuid.New(), srv); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}

	readDone := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		signaled := false
		for {
			n, err := cli.Read(buf)
			if n > 0 && !signaled {
				close(readDone)
				signaled = true
			}
			if err != nil {
				return
			}
		}
	}()

	svc.BroadcastToRoom(channelID, &model.Message{ChannelID: channelID, Content: "x"})
	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("expected write to client despite save error")
	}
}
