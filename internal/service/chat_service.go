package service

import (
	"sync"

	"github.com/Kash4299/todo-chat-app/internal/model"
	chatRepo "github.com/Kash4299/todo-chat-app/internal/repository/chat"
)

type ChatService struct {
	repo  chatRepo.IChatRepository
	mu    sync.RWMutex
	rooms map[string]map[chan *model.ChatMessage]bool
}

func NewChatService(repo chatRepo.IChatRepository) IChatService {
	return &ChatService{
		repo:  repo,
		rooms: make(map[string]map[chan *model.ChatMessage]bool),
	}
}

func (s *ChatService) SaveMessage(msg *model.ChatMessage) error {
	return s.repo.SaveMessage(msg)
}

func (s *ChatService) GetMessagesByRoomID(roomID string) ([]model.ChatMessage, error) {
	return s.repo.GetMessagesByRoomID(roomID)
}

func (s *ChatService) RegisterClient(roomID string, client chan *model.ChatMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rooms[roomID] == nil {
		s.rooms[roomID] = make(map[chan *model.ChatMessage]bool)
	}
	s.rooms[roomID][client] = true
}

func (s *ChatService) UnregisterClient(roomID string, client chan *model.ChatMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clients, ok := s.rooms[roomID]; ok {
		delete(clients, client)
		close(client)
		if len(clients) == 0 {
			delete(s.rooms, roomID)
		}
	}
}

func (s *ChatService) BroadcastToRoom(roomID string, msg *model.ChatMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if clients, ok := s.rooms[roomID]; ok {
		for client := range clients {
			select {
			case client <- msg:
			default:
				// skip slow clients
			}
		}
	}
}
