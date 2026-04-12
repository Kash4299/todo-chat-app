package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatHandler struct {
	service  service.IChatService
	upgrader websocket.Upgrader
}

func NewChatHandler(service service.IChatService) *ChatHandler {
	return &ChatHandler{
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // allow all origins for dev
			},
		},
	}
}

type IncomingMessage struct {
	SenderID uint   `json:"sender_id"`
	Content  string `json:"content"`
}

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	roomID := c.Param("roomID")

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Create a client channel and register
	client := make(chan *model.ChatMessage, 256)
	h.service.RegisterClient(roomID, client)
	defer h.service.UnregisterClient(roomID, client)

	// Goroutine to write messages from the hub to the WebSocket
	go func() {
		for msg := range client {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("websocket write error: %v", err)
				return
			}
		}
	}()

	// Read messages from the WebSocket and broadcast
	for {
		var incoming IncomingMessage
		if err := conn.ReadJSON(&incoming); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("websocket read error: %v", err)
			}
			break
		}

		msg := &model.ChatMessage{
			RoomID:   roomID,
			SenderID: incoming.SenderID,
			Content:  incoming.Content,
		}

		// Persist message
		if err := h.service.SaveMessage(msg); err != nil {
			log.Printf("failed to save message: %v", err)
		}

		// Broadcast to room
		h.service.BroadcastToRoom(roomID, msg)
	}
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	roomID := c.Param("roomID")

	messages, err := h.service.GetMessagesByRoomID(roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": messages})
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	roomID := c.Param("roomID")

	var incoming IncomingMessage
	if err := c.ShouldBindJSON(&incoming); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	senderID, _ := strconv.ParseUint(c.Param("senderID"), 10, 64)
	if incoming.SenderID == 0 {
		incoming.SenderID = uint(senderID)
	}

	msg := &model.ChatMessage{
		RoomID:   roomID,
		SenderID: incoming.SenderID,
		Content:  incoming.Content,
	}

	if err := h.service.SaveMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.service.BroadcastToRoom(roomID, msg)

	c.JSON(http.StatusCreated, gin.H{"data": msg})
}
