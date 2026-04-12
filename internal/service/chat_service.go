package service

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"sync"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/repository/message"
	"github.com/Kash4299/todo-chat-app/internal/repository/taskmember"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
)

type IChatService interface {
	JoinRoom(taskID, userID uuid.UUID, conn net.Conn) error
	LeaveRoom(taskID uuid.UUID, conn net.Conn)
	BroadcastToRoom(taskID uuid.UUID, msg *model.Message)
}
type wsClient struct {
	conn net.Conn
	mu   sync.Mutex
}

type ChatService struct {
	rooms          map[uuid.UUID]map[net.Conn]*wsClient
	mu             sync.RWMutex
	msgRepo        message.IMessageRepository
	taskMemberRepo taskmember.ITaskMemberRepository
}

func NewChatService(msgRepo message.IMessageRepository, taskMemberRepo taskmember.ITaskMemberRepository) IChatService {
	return &ChatService{
		rooms:          make(map[uuid.UUID]map[net.Conn]*wsClient),
		msgRepo:        msgRepo,
		taskMemberRepo: taskMemberRepo,
	}
}

func (s *ChatService) JoinRoom(taskID, userID uuid.UUID, conn net.Conn) error {
	isMember, err := s.taskMemberRepo.IsMember(taskID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("unauthorized: user is not a member")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rooms[taskID] == nil {
		s.rooms[taskID] = make(map[net.Conn]*wsClient)
	}
	s.rooms[taskID][conn] = &wsClient{
		conn: conn,
	}
	return nil
}

func (s *ChatService) LeaveRoom(taskID uuid.UUID, conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.rooms[taskID]; ok {
		delete(s.rooms[taskID], conn)
		if len(s.rooms[taskID]) == 0 {
			delete(s.rooms, taskID)
		}
	}
}

func (s *ChatService) BroadcastToRoom(taskID uuid.UUID, msg *model.Message) {
	if err := s.msgRepo.SaveMessage(msg); err != nil {
		log.Printf("Failed to save message: %v", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	conns, ok := s.rooms[taskID]
	if !ok {
		return
	}

	for _, client := range conns {
		client.mu.Lock()
		jsonData, err := json.Marshal(msg)
		if err == nil {
			err = wsutil.WriteServerText(client.conn, jsonData)
		}
		client.mu.Unlock()

		if err != nil {
			log.Printf("Error writing json to websocket: %v", err)
			client.conn.Close()
		}
	}
}
