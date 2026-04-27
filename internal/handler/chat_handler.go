package handler

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/Kash4299/todo-chat-app/pkg/response"
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
	for _, o := range strings.Split(cfg.AllowedOrigins, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
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

type incomingMessage struct {
	MessageType string `json:"message_type"`
	Content     string `json:"content"`
}

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	channelIDStr := c.Param("channelID")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		response.BadRequest(c, response.CodeInvalidInput, "invalid channel id")
		return
	}

	userIDVal, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		response.Unauthorized(c, response.CodeUnauthorized, "unauthenticated")
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		response.Unauthorized(c, response.CodeUnauthorized, "invalid user identity")
		return
	}

	if !h.isOriginAllowed(c.Request.Header.Get("Origin")) {
		response.Forbidden(c)
		return
	}

	conn, _, _, err := ws.UpgradeHTTP(c.Request, c.Writer)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}

	if err := h.service.JoinRoom(channelID, userID, conn); err != nil {
		conn.Close()
		return
	}
	defer func() {
		h.service.LeaveRoom(channelID, conn)
		conn.Close()
	}()

	for {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		msgData, op, err := wsutil.ReadClientData(conn)
		if err != nil {
			log.Printf("ws read error: %v", err)
			break
		}
		if op != ws.OpText {
			continue
		}

		var incoming incomingMessage
		if err := json.Unmarshal(msgData, &incoming); err != nil {
			log.Printf("ws json parse error: %v", err)
			continue
		}

		msg := &model.Message{
			ChannelID:   channelID,
			UserID:      userID,
			MessageType: incoming.MessageType,
			Content:     incoming.Content,
		}
		h.service.BroadcastToRoom(channelID, msg)
	}
}
