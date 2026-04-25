package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ── Mock: refresh token repo ──────────────────────────────────────────────────

type mockRefreshTokenRepo struct {
	saveErr        error
	storedToken    *model.RefreshToken
	findByHashErr  error
	deleteByHashErr error
}

func (m *mockRefreshTokenRepo) Save(token *model.RefreshToken) error {
	return m.saveErr
}
func (m *mockRefreshTokenRepo) FindByHash(hash string) (*model.RefreshToken, error) {
	return m.storedToken, m.findByHashErr
}
func (m *mockRefreshTokenRepo) DeleteByHash(hash string) error {
	return m.deleteByHashErr
}
func (m *mockRefreshTokenRepo) DeleteAllByUserID(userID uuid.UUID) error {
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newTestLocalAuthSvc(userRepo *mockUserRepo, tokenRepo *mockRefreshTokenRepo) service.ILocalAuthService {
	svc, err := service.NewLocalAuthService(userRepo, tokenRepo, &config.Config{
		JWTSecret:           "test-secret-at-least-32-characters!!",
		JWTAccessExpiryMin:  15,
		JWTRefreshExpiryDay: 7,
	})
	if err != nil {
		panic(err)
	}
	return svc
}

func mustBcrypt(password string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}

// ── Tests: NewLocalAuthService ────────────────────────────────────────────────

func TestLocalAuth_NewService_RejectsEmptySecret(t *testing.T) {
	_, err := service.NewLocalAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{}, &config.Config{JWTSecret: ""})
	if err == nil {
		t.Fatal("expected error for empty JWT_SECRET")
	}
}

// ── Tests: Register ───────────────────────────────────────────────────────────

func TestLocalAuth_Register_Success(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	user, pair, err := svc.Register("Test@Example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user == nil || user.Email != "test@example.com" {
		t.Fatal("expected user with lowercased email")
	}
	if pair == nil || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected non-empty token pair")
	}
	if user.PasswordHash == nil || *user.PasswordHash == "password123" {
		t.Fatal("expected password to be bcrypt-hashed")
	}
}

func TestLocalAuth_Register_DisplayNameFallsBackToEmail(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	user, _, err := svc.Register("a@example.com", "password123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.DisplayName != "a@example.com" {
		t.Fatalf("expected email as display name fallback, got %s", user.DisplayName)
	}
}

func TestLocalAuth_Register_EmailTaken(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailUser: &model.User{ID: uuid.New(), Email: "a@example.com"}}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	_, _, err := svc.Register("a@example.com", "password123", "A")
	if !errors.Is(err, service.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestLocalAuth_Register_PasswordTooShort(t *testing.T) {
	svc := newTestLocalAuthSvc(&mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}, &mockRefreshTokenRepo{})

	_, _, err := svc.Register("a@example.com", "short", "A")
	if !errors.Is(err, service.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestLocalAuth_Register_FindByEmailRepoError(t *testing.T) {
	expected := errors.New("db error")
	svc := newTestLocalAuthSvc(&mockUserRepo{findByEmailErr: expected}, &mockRefreshTokenRepo{})

	_, _, err := svc.Register("a@example.com", "password123", "A")
	if !errors.Is(err, expected) {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestLocalAuth_Register_CreateUserError(t *testing.T) {
	expected := errors.New("create failed")
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound, createErr: expected}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	_, _, err := svc.Register("a@example.com", "password123", "A")
	if !errors.Is(err, expected) {
		t.Fatalf("expected create error, got %v", err)
	}
}

// ── Tests: Login ──────────────────────────────────────────────────────────────

func TestLocalAuth_Login_Success(t *testing.T) {
	hash := mustBcrypt("password123")
	userRepo := &mockUserRepo{findByEmailUser: &model.User{ID: uuid.New(), Email: "a@example.com", PasswordHash: &hash}}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	user, pair, err := svc.Login("a@example.com", "password123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user == nil || pair == nil {
		t.Fatal("expected user and token pair")
	}
}

func TestLocalAuth_Login_WrongPassword(t *testing.T) {
	hash := mustBcrypt("correct-password")
	userRepo := &mockUserRepo{findByEmailUser: &model.User{ID: uuid.New(), Email: "a@example.com", PasswordHash: &hash}}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	_, _, err := svc.Login("a@example.com", "wrong-password")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLocalAuth_Login_NoPasswordSet(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailUser: &model.User{ID: uuid.New(), Email: "a@example.com"}}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	_, _, err := svc.Login("a@example.com", "anything")
	if !errors.Is(err, service.ErrNoPasswordSet) {
		t.Fatalf("expected ErrNoPasswordSet, got %v", err)
	}
}

func TestLocalAuth_Login_NotFound(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	_, _, err := svc.Login("a@example.com", "password")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

// ── Tests: Refresh ────────────────────────────────────────────────────────────

func TestLocalAuth_Refresh_Success(t *testing.T) {
	stored := &model.RefreshToken{
		UserID:    uuid.New(),
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	tokenRepo := &mockRefreshTokenRepo{storedToken: stored}
	svc := newTestLocalAuthSvc(&mockUserRepo{}, tokenRepo)

	pair, err := svc.Refresh("any-raw-token")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pair == nil || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected non-empty token pair")
	}
}

func TestLocalAuth_Refresh_NotFound(t *testing.T) {
	tokenRepo := &mockRefreshTokenRepo{findByHashErr: gorm.ErrRecordNotFound}
	svc := newTestLocalAuthSvc(&mockUserRepo{}, tokenRepo)

	_, err := svc.Refresh("invalid-token")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLocalAuth_Refresh_Expired(t *testing.T) {
	stored := &model.RefreshToken{
		UserID:    uuid.New(),
		TokenHash: "hash",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	tokenRepo := &mockRefreshTokenRepo{storedToken: stored}
	svc := newTestLocalAuthSvc(&mockUserRepo{}, tokenRepo)

	_, err := svc.Refresh("old-token")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for expired token, got %v", err)
	}
}

// ── Tests: Logout ─────────────────────────────────────────────────────────────

func TestLocalAuth_Logout_AlwaysSucceeds(t *testing.T) {
	svc := newTestLocalAuthSvc(&mockUserRepo{}, &mockRefreshTokenRepo{})
	if err := svc.Logout("any-token"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ── Tests: SetPassword ────────────────────────────────────────────────────────

func TestLocalAuth_SetPassword_Success(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepo{findByIDUser: &model.User{ID: userID, Email: "a@example.com"}}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	if err := svc.SetPassword(userID, "newpassword"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if userRepo.updateCalls != 1 {
		t.Fatalf("expected one update call, got %d", userRepo.updateCalls)
	}
}

func TestLocalAuth_SetPassword_AlreadySet(t *testing.T) {
	hash := "existinghash"
	userRepo := &mockUserRepo{findByIDUser: &model.User{ID: uuid.New(), PasswordHash: &hash}}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	if err := svc.SetPassword(uuid.New(), "newpassword"); !errors.Is(err, service.ErrPasswordAlreadySet) {
		t.Fatalf("expected ErrPasswordAlreadySet, got %v", err)
	}
}

func TestLocalAuth_SetPassword_TooShort(t *testing.T) {
	userRepo := &mockUserRepo{findByIDUser: &model.User{ID: uuid.New()}}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	if err := svc.SetPassword(uuid.New(), "short"); !errors.Is(err, service.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}
