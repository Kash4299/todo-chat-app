package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockWorkspaceRoleService struct {
	role       string
	err        error
	callCount  int
	lastWSID   uuid.UUID
	lastUserID uuid.UUID
}

func (m *mockWorkspaceRoleService) GetRole(workspaceID, userID uuid.UUID) (string, error) {
	m.callCount++
	m.lastWSID = workspaceID
	m.lastUserID = userID
	if m.err != nil {
		return "", m.err
	}
	return m.role, nil
}

type rbacTestHarness struct {
	router      *gin.Engine
	roleService *mockWorkspaceRoleService
}

func newRBACTestHarness(t *testing.T) *rbacTestHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	roleService := &mockWorkspaceRoleService{}
	rbac := middleware.NewRBACMiddleware(roleService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID := c.GetHeader("X-UserID"); userID != "" {
			if parsed, err := uuid.Parse(userID); err == nil {
				c.Set(middleware.UserIDContextKey, parsed)
			}
		}
		c.Next()
	})
	r.GET("/workspaces/:workspaceID/resource", rbac.RequireWorkspaceRole([]string{middleware.AdminRole, middleware.MemberRole, middleware.GuestRole}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	return &rbacTestHarness{router: r, roleService: roleService}
}

func (h *rbacTestHarness) doRequest(t *testing.T, workspaceID, userID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID+"/resource", nil)
	if userID != "" {
		req.Header.Set("X-UserID", userID)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

func assertStatusCode(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Fatalf("expected status %d, got %d, body=%s", expected, w.Code, w.Body.String())
	}
}

func assertServiceNotCalled(t *testing.T, h *rbacTestHarness) {
	t.Helper()
	if h.roleService.callCount != 0 {
		t.Fatalf("expected GetRole call count 0, got %d", h.roleService.callCount)
	}
}

func assertServiceCalledOnceWith(t *testing.T, h *rbacTestHarness, expectedWorkspaceID, expectedUserID uuid.UUID) {
	t.Helper()
	if h.roleService.callCount != 1 {
		t.Fatalf("expected GetRole call count 1, got %d", h.roleService.callCount)
	}
	if h.roleService.lastWSID != expectedWorkspaceID {
		t.Fatalf("expected workspaceID %s, got %s", expectedWorkspaceID, h.roleService.lastWSID)
	}
	if h.roleService.lastUserID != expectedUserID {
		t.Fatalf("expected userID %s, got %s", expectedUserID, h.roleService.lastUserID)
	}
}

func TestRBACMiddleware_RequireWorkspaceRole_HTTPFlow(t *testing.T) {
	tests := []struct {
		name             string
		workspaceID      string
		userID           string
		serviceRole      string
		serviceErr       error
		expectedStatus   int
		expectSvcCalled  bool
		expectedSvcCalls int
	}{
		{
			name:             "rejects request when user context is missing",
			workspaceID:      uuid.NewString(),
			userID:           "",
			expectedStatus:   http.StatusUnauthorized,
			expectSvcCalled:  false,
			expectedSvcCalls: 0,
		},
		{
			name:             "rejects request when workspaceID is invalid UUID",
			workspaceID:      "not-a-uuid",
			userID:           uuid.NewString(),
			expectedStatus:   http.StatusBadRequest,
			expectSvcCalled:  false,
			expectedSvcCalls: 0,
		},
		{
			name:             "rejects request when userID context is not UUID",
			workspaceID:      uuid.NewString(),
			userID:           "not-a-uuid",
			expectedStatus:   http.StatusUnauthorized,
			expectSvcCalled:  false,
			expectedSvcCalls: 0,
		},
		{
			name:             "returns forbidden when role service returns record not found",
			workspaceID:      uuid.NewString(),
			userID:           uuid.NewString(),
			serviceErr:       gorm.ErrRecordNotFound,
			expectedStatus:   http.StatusForbidden,
			expectSvcCalled:  true,
			expectedSvcCalls: 1,
		},
		{
			name:             "returns internal server error when role service returns system error",
			workspaceID:      uuid.NewString(),
			userID:           uuid.NewString(),
			serviceErr:       errors.New("db timeout"),
			expectedStatus:   http.StatusInternalServerError,
			expectSvcCalled:  true,
			expectedSvcCalls: 1,
		},
		{
			name:             "rejects request when role is outside allowed roles",
			workspaceID:      uuid.NewString(),
			userID:           uuid.NewString(),
			serviceRole:      "VIEWER",
			expectedStatus:   http.StatusForbidden,
			expectSvcCalled:  true,
			expectedSvcCalls: 1,
		},
		{
			name:             "allows request when role is ADMIN",
			workspaceID:      uuid.NewString(),
			userID:           uuid.NewString(),
			serviceRole:      middleware.AdminRole,
			expectedStatus:   http.StatusOK,
			expectSvcCalled:  true,
			expectedSvcCalls: 1,
		},
		{
			name:             "allows request when role is MEMBER",
			workspaceID:      uuid.NewString(),
			userID:           uuid.NewString(),
			serviceRole:      middleware.MemberRole,
			expectedStatus:   http.StatusOK,
			expectSvcCalled:  true,
			expectedSvcCalls: 1,
		},
		{
			name:             "allows request when role is GUEST",
			workspaceID:      uuid.NewString(),
			userID:           uuid.NewString(),
			serviceRole:      middleware.GuestRole,
			expectedStatus:   http.StatusOK,
			expectSvcCalled:  true,
			expectedSvcCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newRBACTestHarness(t)
			h.roleService.role = tt.serviceRole
			h.roleService.err = tt.serviceErr

			w := h.doRequest(t, tt.workspaceID, tt.userID)
			assertStatusCode(t, w, tt.expectedStatus)

			if h.roleService.callCount != tt.expectedSvcCalls {
				t.Fatalf("expected GetRole call count %d, got %d", tt.expectedSvcCalls, h.roleService.callCount)
			}

			if tt.expectSvcCalled {
				expectedWorkspaceUUID, err := uuid.Parse(tt.workspaceID)
				if err != nil {
					t.Fatalf("test setup error: workspaceID must be valid UUID when expectSvcCalled=true: %v", err)
				}
				expectedUserUUID, err := uuid.Parse(tt.userID)
				if err != nil {
					t.Fatalf("test setup error: userID must be valid UUID when expectSvcCalled=true: %v", err)
				}
				assertServiceCalledOnceWith(t, h, expectedWorkspaceUUID, expectedUserUUID)
			} else {
				assertServiceNotCalled(t, h)
			}
		})
	}
}

func TestRBACMiddleware_WorkspaceIDRequired_DirectInvocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	roleService := &mockWorkspaceRoleService{}
	rbac := middleware.NewRBACMiddleware(roleService)
	handler := rbac.RequireWorkspaceRole([]string{middleware.AdminRole})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(middleware.UserIDContextKey, uuid.New())

	handler(c)

	assertStatusCode(t, w, http.StatusBadRequest)
	if roleService.callCount != 0 {
		t.Fatalf("expected GetRole call count 0, got %d", roleService.callCount)
	}
}

func TestRBACMiddleware_UserContextWrongType_ReturnsUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	roleService := &mockWorkspaceRoleService{}
	rbac := middleware.NewRBACMiddleware(roleService)
	handler := rbac.RequireWorkspaceRole([]string{middleware.AdminRole})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "workspaceID", Value: uuid.NewString()}}
	c.Set(middleware.UserIDContextKey, 12345)

	handler(c)
	assertStatusCode(t, w, http.StatusUnauthorized)
	if roleService.callCount != 0 {
		t.Fatalf("expected GetRole call count 0, got %d", roleService.callCount)
	}
}
