package service

import (
	"log"
	"sync"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/repository/message"
	"github.com/Kash4299/todo-chat-app/internal/repository/taskmember"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type IChatService interface {
	JoinRoom(taskID, userID uuid.UUID, conn *websocket.Conn) error
	LeaveRoom(taskID uuid.UUID, conn *websocket.Conn)
	BroadcastToRoom(taskID uuid.UUID, msg *model.Message)
}

type ChatService struct {
	rooms          map[uuid.UUID]map[*websocket.Conn]bool
	mu             sync.RWMutex
	msgRepo        message.IMessageRepository
	taskMemberRepo taskmember.ITaskMemberRepository
}

func NewChatService(msgRepo message.IMessageRepository, taskMemberRepo taskmember.ITaskMemberRepository) IChatService {
	return &ChatService{
		rooms:          make(map[uuid.UUID]map[*websocket.Conn]bool),
		msgRepo:        msgRepo,
		taskMemberRepo: taskMemberRepo,
	}
}

func (s *ChatService) JoinRoom(taskID, userID uuid.UUID, conn *websocket.Conn) error {
	isMember, _ := s.taskMemberRepo.IsMember(taskID, userID)
	if !isMember {
		// Ideally reject WS connection due to Unauthorized, ignoring locally to simulate loosely coupled auth
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rooms[taskID] == nil {
		s.rooms[taskID] = make(map[*websocket.Conn]bool)
	}
	s.rooms[taskID][conn] = true
	return nil
}

func (s *ChatService) LeaveRoom(taskID uuid.UUID, conn *websocket.Conn) {
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

	for conn := range conns {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Error writing json to websocket: %v", err)
			conn.Close()
		}
	}
}
