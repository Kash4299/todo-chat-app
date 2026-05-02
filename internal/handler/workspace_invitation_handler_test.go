package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockWorkspaceInvitationService struct {
	inviteErr error
	resendErr error
	listErr   error
	acceptErr error
}

func (m *mockWorkspaceInvitationService) Invite(workspaceID, inviterID uuid.UUID, inviteeEmail string) error {
	return m.inviteErr
}

func (m *mockWorkspaceInvitationService) ResendInvitation(workspaceID, inviterID uuid.UUID, inviteeEmail string) error {
	return m.resendErr
}

func (m *mockWorkspaceInvitationService) GetListInvitations(workspaceID uuid.UUID) ([]model.WorkspaceInvitation, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []model.WorkspaceInvitation{}, nil
}

func (m *mockWorkspaceInvitationService) AcceptInvitation(actorID uuid.UUID, token string) (uuid.UUID, error) {
	return uuid.Nil, m.acceptErr
}

func TestWorkspaceInvitationHandler_InviteRejectsWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWorkspaceInvitationHandler(&mockWorkspaceInvitationService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+uuid.NewString()+"/invite", bytes.NewReader([]byte(`{"email":"a@b.com"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "workspaceID", Value: uuid.NewString()}}

	h.Invite(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestWorkspaceInvitationHandler_InviteMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{name: "not admin", serviceErr: constants.ErrWorkspaceInvitationNotAdmin, wantStatus: http.StatusForbidden},
		{name: "already member", serviceErr: constants.ErrWorkspaceInvitationIsMember, wantStatus: http.StatusBadRequest},
		{name: "still valid", serviceErr: constants.ErrWorkspaceInvitationStillValid, wantStatus: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewWorkspaceInvitationHandler(&mockWorkspaceInvitationService{inviteErr: tc.serviceErr})
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+uuid.NewString()+"/invite", bytes.NewReader([]byte(`{"email":"a@b.com"}`)))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "workspaceID", Value: uuid.NewString()}}
			c.Set(middleware.UserIDContextKey, uuid.New())

			h.Invite(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestWorkspaceInvitationHandler_ResendMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{name: "not admin", serviceErr: constants.ErrWorkspaceInvitationNotAdmin, wantStatus: http.StatusForbidden},
		{name: "already member", serviceErr: constants.ErrWorkspaceInvitationIsMember, wantStatus: http.StatusBadRequest},
		{name: "still valid", serviceErr: constants.ErrWorkspaceInvitationStillValid, wantStatus: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewWorkspaceInvitationHandler(&mockWorkspaceInvitationService{resendErr: tc.serviceErr})
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+uuid.NewString()+"/resend-invite", bytes.NewReader([]byte(`{"email":"a@b.com"}`)))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "workspaceID", Value: uuid.NewString()}}
			c.Set(middleware.UserIDContextKey, uuid.New())

			h.ResendInvitation(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestWorkspaceInvitationHandler_ResendRejectsWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWorkspaceInvitationHandler(&mockWorkspaceInvitationService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+uuid.NewString()+"/resend-invite", bytes.NewReader([]byte(`{"email":"a@b.com"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "workspaceID", Value: uuid.NewString()}}

	h.ResendInvitation(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestWorkspaceInvitationHandler_AcceptRejectsWithoutAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWorkspaceInvitationHandler(&mockWorkspaceInvitationService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/invitations/accept", bytes.NewReader([]byte(`{"token":"t"}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.AcceptInvitation(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestWorkspaceInvitationHandler_AcceptMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{name: "not found", serviceErr: gorm.ErrRecordNotFound, wantStatus: http.StatusNotFound},
		{name: "already member", serviceErr: constants.ErrWorkspaceInvitationIsMember, wantStatus: http.StatusBadRequest},
		{name: "expired", serviceErr: constants.ErrWorkspaceInvitationIsExpired, wantStatus: http.StatusBadRequest},
		{name: "email mismatch", serviceErr: constants.ErrWorkspaceInvitationEmailMismatch, wantStatus: http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewWorkspaceInvitationHandler(&mockWorkspaceInvitationService{acceptErr: tc.serviceErr})
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/invitations/accept", bytes.NewReader([]byte(`{"token":"t"}`)))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set(middleware.UserIDContextKey, uuid.New())

			h.AcceptInvitation(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestWorkspaceInvitationHandler_GetListInvalidWorkspaceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWorkspaceInvitationHandler(&mockWorkspaceInvitationService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/invalid/invitations", nil)
	c.Params = gin.Params{{Key: "workspaceID", Value: "invalid"}}

	h.GetListInvitations(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
