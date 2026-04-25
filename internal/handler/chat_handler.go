package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
)

type ChatHandler struct {
	service        service.IChatService
	allowedOrigins []string
}

func NewChatHandler(svc service.IChatService, cfg *config.Config) *ChatHandler {
	var origins []string
	if cfg.AllowedOrigins != "" {
		for _, o := range strings.Split(cfg.AllowedOrigins, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	}
	return &ChatHandler{service: svc, allowedOrigins: origins}
}

func (h *ChatHandler) isOriginAllowed(origin string) bool {
	if len(h.allowedOrigins) == 0 {
		return true
	}
	for _, allowed := range h.allowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}

type IncomingMessage struct {
	MessageType string `json:"message_type"`
	Content     string `json:"content"`
}

func messageFromIncoming(taskID, userID uuid.UUID, incoming IncomingMessage) *model.Message {
	return &model.Message{
		TaskID:      taskID,
		UserID:      userID,
		MessageType: incoming.MessageType,
		Content:     incoming.Content,
	}
}

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	taskIDStr := c.Param("taskID")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID format"})
		return
	}

	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user identity in context"})
		return
	}

	if !h.isOriginAllowed(c.Request.Header.Get("Origin")) {
		c.JSON(http.StatusForbidden, gin.H{"error": "origin not allowed"})
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
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
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

		msg := messageFromIncoming(taskID, userID, incoming)
		h.service.BroadcastToRoom(taskID, msg)
	}
}
