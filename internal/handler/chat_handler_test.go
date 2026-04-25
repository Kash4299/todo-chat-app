package handler

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockChatService struct{}

func (m *mockChatService) JoinRoom(_ uuid.UUID, _ uuid.UUID, _ net.Conn) error { return nil }
func (m *mockChatService) LeaveRoom(_ uuid.UUID, _ net.Conn)                   {}
func (m *mockChatService) BroadcastToRoom(_ uuid.UUID, _ *model.Message)       {}

func TestHandleWebSocket_RejectsRequestWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewChatHandler(&mockChatService{}, &config.Config{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/ws/channels/"+uuid.New().String(), nil)
	c.Params = gin.Params{{Key: "channelID", Value: uuid.New().String()}}
	// UserIDContextKey intentionally not set → should return 401

	h.HandleWebSocket(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleWebSocket_RejectsInvalidChannelID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewChatHandler(&mockChatService{}, &config.Config{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/ws/channels/not-a-uuid", nil)
	c.Params = gin.Params{{Key: "channelID", Value: "not-a-uuid"}}
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.HandleWebSocket(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
