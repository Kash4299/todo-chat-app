package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
)

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

	conn, _, _, err := ws.UpgradeHTTP(c.Request, c.Writer)
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
		msgData, op, err := wsutil.ReadClientData(conn)
		if err != nil {
			log.Printf("error reading websocket stream: %v", err)
			break
		}

		if op != ws.OpText {
			continue // only accepting text payload containing json
		}

		var incoming IncomingMessage
		if err := json.Unmarshal(msgData, &incoming); err != nil {
			log.Printf("error parsing json: %v", err)
			continue
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
