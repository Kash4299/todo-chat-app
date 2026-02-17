package services

import (
	"context"
	"sync"
	"time"
	"todo/internal/model"

	"github.com/google/uuid"
)

// ChatService manages chat rooms and message persistence
type ChatService struct {
	hub *Hub

	mu    sync.RWMutex
	rooms map[string]map[string]chan *model.ChatMessage // room -> clientID -> channel
}

func NewChatService(hub *Hub) *ChatService {
	return &ChatService{
		hub:   hub,
		rooms: make(map[string]map[string]chan *model.ChatMessage),
	}
}

// SendMessage persists a message and broadcasts it to all clients in the room
func (s *ChatService) SendMessage(ctx context.Context, room, sender, content string) (*model.ChatMessage, error) {
	senderUUID, err := uuid.Parse(sender)
	if err != nil {
		return nil, err
	}

	msg := &model.ChatMessage{
		BaseModel: model.BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Room:    room,
		Sender:  senderUUID,
		Content: content,
	}

	// Persist to database
	if err := s.hub.ChatRepository.SaveMessage(ctx, msg); err != nil {
		return nil, err
	}

	// Broadcast to all connected clients in the room
	s.broadcast(room, msg)

	return msg, nil
}

// Join adds a client to a room and returns a channel to receive messages
func (s *ChatService) Join(room, clientID string) <-chan *model.ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rooms[room] == nil {
		s.rooms[room] = make(map[string]chan *model.ChatMessage)
	}

	ch := make(chan *model.ChatMessage, 64)
	s.rooms[room][clientID] = ch
	return ch
}

// Leave removes a client from a room
func (s *ChatService) Leave(room, clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clients, ok := s.rooms[room]; ok {
		if ch, exists := clients[clientID]; exists {
			close(ch)
			delete(clients, clientID)
		}
		if len(clients) == 0 {
			delete(s.rooms, room)
		}
	}
}

// broadcast sends a message to all clients in a room
func (s *ChatService) broadcast(room string, msg *model.ChatMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if clients, ok := s.rooms[room]; ok {
		for _, ch := range clients {
			select {
			case ch <- msg:
			default:
				// channel full, skip to avoid blocking
			}
		}
	}
}

// GetRoomHistory retrieves past messages for a room
func (s *ChatService) GetRoomHistory(ctx context.Context, room string, limit int) (*[]model.ChatMessage, error) {
	return s.hub.ChatRepository.GetMessagesByRoom(ctx, room, limit)
}
