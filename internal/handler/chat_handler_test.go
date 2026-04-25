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

func TestMessageFromIncomingMapsFields(t *testing.T) {
	taskID := uuid.New()
	userID := uuid.New()

	msg := messageFromIncoming(taskID, userID, IncomingMessage{
		MessageType: "TEXT",
		Content:     "hello",
	})

	if msg.UserID != userID {
		t.Fatalf("expected user ID %s, got %s", userID, msg.UserID)
	}
	if msg.TaskID != taskID {
		t.Fatalf("expected task ID %s, got %s", taskID, msg.TaskID)
	}
	if msg.Content != "hello" || msg.MessageType != "TEXT" {
		t.Fatalf("unexpected message contents: %+v", msg)
	}
}

func TestHandleWebSocket_RejectsRequestWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewChatHandler(&mockChatService{}, &config.Config{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/ws/chat/"+uuid.New().String(), nil)
	c.Params = gin.Params{{Key: "taskID", Value: uuid.New().String()}}
	// middleware.UserIDContextKey is intentionally not set

	h.HandleWebSocket(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleWebSocket_RejectsInvalidTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewChatHandler(&mockChatService{}, &config.Config{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/ws/chat/not-a-uuid", nil)
	c.Params = gin.Params{{Key: "taskID", Value: "not-a-uuid"}}
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.HandleWebSocket(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
