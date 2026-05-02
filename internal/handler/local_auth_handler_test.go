package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/middleware"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── Mock ──────────────────────────────────────────────────────────────────────

type mockLocalAuthService struct {
	registerErr    error
	resendErr      error
	loginErr       error
	refreshErr     error
	setPasswordErr error
	confirmLinkErr error
	verifyEmailErr error
	user           *model.User
	pair           *service.TokenPair
}

func (m *mockLocalAuthService) Register(email, password, displayName string) (*model.User, error) {
	if m.registerErr != nil {
		return nil, m.registerErr
	}
	u := m.user
	if u == nil {
		u = &model.User{ID: uuid.New(), Email: email}
	}
	return u, nil
}

func (m *mockLocalAuthService) VerifyEmail(rawToken string) (*model.User, *service.TokenPair, error) {
	if m.verifyEmailErr != nil {
		return nil, nil, m.verifyEmailErr
	}
	u := m.user
	if u == nil {
		u = &model.User{ID: uuid.New()}
	}
	p := m.pair
	if p == nil {
		p = &service.TokenPair{AccessToken: "access", RefreshToken: "refresh"}
	}
	return u, p, nil
}

func (m *mockLocalAuthService) ResendVerification(email string) error {
	return m.resendErr
}

func (m *mockLocalAuthService) Login(email, password string) (*model.User, *service.TokenPair, error) {
	if m.loginErr != nil {
		return nil, nil, m.loginErr
	}
	u := m.user
	if u == nil {
		u = &model.User{ID: uuid.New(), Email: email}
	}
	p := m.pair
	if p == nil {
		p = &service.TokenPair{AccessToken: "access", RefreshToken: "refresh"}
	}
	return u, p, nil
}

func (m *mockLocalAuthService) Refresh(rawRefreshToken string) (*service.TokenPair, error) {
	if m.refreshErr != nil {
		return nil, m.refreshErr
	}
	p := m.pair
	if p == nil {
		p = &service.TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh"}
	}
	return p, nil
}

func (m *mockLocalAuthService) Logout(rawRefreshToken string) error { return nil }

func (m *mockLocalAuthService) SetPassword(userID uuid.UUID, newPassword string) error {
	return m.setPasswordErr
}

func (m *mockLocalAuthService) ConfirmAccountLink(pendingToken, password string) (*model.User, *service.TokenPair, error) {
	if m.confirmLinkErr != nil {
		return nil, nil, m.confirmLinkErr
	}
	u := m.user
	if u == nil {
		u = &model.User{ID: uuid.New()}
	}
	p := m.pair
	if p == nil {
		p = &service.TokenPair{AccessToken: "access", RefreshToken: "refresh"}
	}
	return u, p, nil
}

// Compile-time: ensure mock satisfies interface
var _ service.ILocalAuthService = (*mockLocalAuthService)(nil)

// Ensure errors package is used
var _ = errors.New

// ── Tests: Register ───────────────────────────────────────────────────────────

func TestLocalAuthHandler_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	body := []byte(`{"email":"a@example.com","password":"password123","display_name":"A"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Register(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLocalAuthHandler_Register_EmailTaken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{registerErr: constants.ErrEmailTaken})

	w := httptest.NewRecorder()
	body := []byte(`{"email":"a@example.com","password":"password123"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Register(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestLocalAuthHandler_Register_PasswordTooShort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{registerErr: constants.ErrPasswordTooShort})

	w := httptest.NewRecorder()
	body := []byte(`{"email":"a@example.com","password":"short"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Register(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLocalAuthHandler_Register_BadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte(`not-json`)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Register(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ── Tests: VerifyEmail ────────────────────────────────────────────────────────

func TestLocalAuthHandler_VerifyEmail_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	body := []byte(`{"token":"valid-raw-token"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.VerifyEmail(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLocalAuthHandler_VerifyEmail_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{verifyEmailErr: constants.ErrInvalidVerificationToken})

	w := httptest.NewRecorder()
	body := []byte(`{"token":"bad-token"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.VerifyEmail(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLocalAuthHandler_VerifyEmail_BadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewReader([]byte(`not-json`)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.VerifyEmail(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLocalAuthHandler_VerifyEmail_BodyTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	oversized := `{"token":"` + strings.Repeat("a", 1<<20) + `"}`
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewReader([]byte(oversized)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.VerifyEmail(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized body, got %d", w.Code)
	}
}

// ── Tests: ResendVerification ────────────────────────────────────────────────

func TestLocalAuthHandler_ResendVerification_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	body := []byte(`{"email":"a@example.com"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ResendVerification(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLocalAuthHandler_ResendVerification_BadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", bytes.NewReader([]byte(`not-json`)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ResendVerification(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLocalAuthHandler_ResendVerification_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{resendErr: errors.New("smtp failed")})

	w := httptest.NewRecorder()
	body := []byte(`{"email":"a@example.com"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ResendVerification(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestLocalAuthHandler_ResendVerification_RateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{resendErr: constants.ErrVerificationEmailRateLimited})

	w := httptest.NewRecorder()
	body := []byte(`{"email":"a@example.com"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ResendVerification(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ── Tests: Login ──────────────────────────────────────────────────────────────

func TestLocalAuthHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	body := []byte(`{"email":"a@example.com","password":"password123"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Login(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLocalAuthHandler_Login_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{loginErr: constants.ErrInvalidCredentials})

	w := httptest.NewRecorder()
	body := []byte(`{"email":"a@example.com","password":"wrong"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Login(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLocalAuthHandler_Login_NoPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{loginErr: constants.ErrNoPasswordSet})

	w := httptest.NewRecorder()
	body := []byte(`{"email":"a@example.com","password":"anything"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Login(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLocalAuthHandler_Login_EmailNotVerified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{loginErr: constants.ErrEmailNotVerified})

	w := httptest.NewRecorder()
	body := []byte(`{"email":"a@example.com","password":"password123"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Login(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// ── Tests: Refresh ────────────────────────────────────────────────────────────

func TestLocalAuthHandler_Refresh_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	body := []byte(`{"refresh_token":"some-token"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Refresh(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLocalAuthHandler_Refresh_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{refreshErr: constants.ErrInvalidCredentials})

	w := httptest.NewRecorder()
	body := []byte(`{"refresh_token":"bad-token"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Refresh(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── Tests: Logout ─────────────────────────────────────────────────────────────

func TestLocalAuthHandler_Logout_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	body := []byte(`{"refresh_token":"some-token"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Logout(c)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

// ── Tests: SetPassword ────────────────────────────────────────────────────────

func TestLocalAuthHandler_SetPassword_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	body := []byte(`{"password":"newpassword"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users/me/password", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.SetPassword(c)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLocalAuthHandler_SetPassword_AlreadySet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{setPasswordErr: constants.ErrPasswordAlreadySet})

	w := httptest.NewRecorder()
	body := []byte(`{"password":"newpassword"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users/me/password", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.UserIDContextKey, uuid.New())

	h.SetPassword(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestLocalAuthHandler_SetPassword_NoAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users/me/password", bytes.NewReader([]byte(`{"password":"x"}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.SetPassword(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── Tests: ConfirmAccountLink ─────────────────────────────────────────────────

func TestLocalAuthHandler_ConfirmAccountLink_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	body := []byte(`{"pending_token":"some-token","password":"password123"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/link/confirm", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ConfirmAccountLink(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLocalAuthHandler_ConfirmAccountLink_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{confirmLinkErr: constants.ErrInvalidPendingToken})

	w := httptest.NewRecorder()
	body := []byte(`{"pending_token":"bad","password":"password123"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/link/confirm", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ConfirmAccountLink(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLocalAuthHandler_ConfirmAccountLink_WrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{confirmLinkErr: constants.ErrInvalidCredentials})

	w := httptest.NewRecorder()
	body := []byte(`{"pending_token":"tok","password":"wrong"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/link/confirm", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ConfirmAccountLink(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLocalAuthHandler_ConfirmAccountLink_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{confirmLinkErr: constants.ErrLinkConflict})

	w := httptest.NewRecorder()
	body := []byte(`{"pending_token":"tok","password":"password123"}`)
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/link/confirm", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ConfirmAccountLink(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestLocalAuthHandler_ConfirmAccountLink_BadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewLocalAuthHandler(&mockLocalAuthService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/link/confirm", bytes.NewReader([]byte(`not-json`)))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ConfirmAccountLink(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
