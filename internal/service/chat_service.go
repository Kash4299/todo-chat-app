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
	JoinRoom(channelID, userID uuid.UUID, conn net.Conn) error
	LeaveRoom(channelID uuid.UUID, conn net.Conn)
	BroadcastToRoom(channelID uuid.UUID, msg *model.Message)
}

type wsClient struct {
	conn net.Conn
	mu   sync.Mutex
}

type roomConn struct {
	conn   net.Conn
	client *wsClient
}

// ChatService is an in-memory single-instance implementation.
// TODO (PoC Spike): replace BroadcastToRoom with Kafka AsyncProducer fan-out
// so that multiple WS server instances can share messages across the cluster.
type ChatService struct {
	rooms         map[uuid.UUID]map[net.Conn]*wsClient
	mu            sync.RWMutex
	msgRepo       message.IMessageRepository
	channelMember taskmember.IChannelMemberRepository
}

func NewChatService(
	msgRepo message.IMessageRepository,
	channelMember taskmember.IChannelMemberRepository,
) IChatService {
	return &ChatService{
		rooms:         make(map[uuid.UUID]map[net.Conn]*wsClient),
		msgRepo:       msgRepo,
		channelMember: channelMember,
	}
}

func (s *ChatService) JoinRoom(channelID, userID uuid.UUID, conn net.Conn) error {
	ok, err := s.channelMember.IsMember(channelID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("unauthorized: user is not a channel member")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rooms[channelID] == nil {
		s.rooms[channelID] = make(map[net.Conn]*wsClient)
	}
	s.rooms[channelID][conn] = &wsClient{conn: conn}
	return nil
}

func (s *ChatService) LeaveRoom(channelID uuid.UUID, conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if room, ok := s.rooms[channelID]; ok {
		delete(room, conn)
		if len(room) == 0 {
			delete(s.rooms, channelID)
		}
	}
}

func (s *ChatService) BroadcastToRoom(channelID uuid.UUID, msg *model.Message) {
	if err := s.msgRepo.Save(msg); err != nil {
		log.Printf("failed to persist message: %v", err)
	}

	s.mu.RLock()
	conns, ok := s.rooms[channelID]
	if !ok {
		s.mu.RUnlock()
		return
	}
	clients := make([]roomConn, 0, len(conns))
	for conn, client := range conns {
		clients = append(clients, roomConn{conn: conn, client: client})
	}
	s.mu.RUnlock()

	var failed []net.Conn
	for _, rc := range clients {
		rc.client.mu.Lock()
		data, err := json.Marshal(msg)
		if err == nil {
			err = wsutil.WriteServerText(rc.client.conn, data)
		}
		rc.client.mu.Unlock()
		if err != nil {
			log.Printf("ws write error: %v", err)
			rc.client.conn.Close()
			failed = append(failed, rc.conn)
		}
	}

	if len(failed) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if room, ok := s.rooms[channelID]; ok {
		for _, conn := range failed {
			delete(room, conn)
		}
		if len(room) == 0 {
			delete(s.rooms, channelID)
		}
	}
}
