package handler

import (
	"log"
	"net/http"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ChatHandler struct {
	service service.IChatService
}

func NewChatHandler(service service.IChatService) *ChatHandler {
	return &ChatHandler{service: service}
}

type IncomingMessage struct {
	SenderID    string `json:"sender_id"`
	MessageType string `json:"message_type"`
	Content     string `json:"content"`
}

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	taskIDStr := c.Param("taskID")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID format"})
		return
	}

	userIDStr := c.Query("userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid userID format in query, mandatory to establish auth validity"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to set websocket upgrade: %+v", err)
		return
	}

	if err := h.service.JoinRoom(taskID, userID, conn); err != nil {
		conn.Close()
		return
	}

	defer func() {
		h.service.LeaveRoom(taskID, conn)
		conn.Close()
	}()

	for {
		var incoming IncomingMessage
		if err := conn.ReadJSON(&incoming); err != nil {
			log.Printf("error reading json: %v", err)
			break
		}

		senderUUID, parseErr := uuid.Parse(incoming.SenderID)
		if parseErr != nil {
			senderUUID = userID // default to connection user
		}

		msg := &model.Message{
			TaskID:      taskID,
			UserID:      senderUUID,
			MessageType: incoming.MessageType,
			Content:     incoming.Content,
		}

		h.service.BroadcastToRoom(taskID, msg)
	}
}
