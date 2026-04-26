package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/golang-jwt/jwt/v5"
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

const testJWTSecret = "test-secret-at-least-32-characters!!"

func newTestLocalAuthSvc(userRepo *mockUserRepo, tokenRepo *mockRefreshTokenRepo) service.ILocalAuthService {
	return newTestLocalAuthSvcFull(userRepo, tokenRepo, &mockUserIdentityRepo{})
}

func newTestLocalAuthSvcFull(userRepo *mockUserRepo, tokenRepo *mockRefreshTokenRepo, identityRepo *mockUserIdentityRepo) service.ILocalAuthService {
	svc, err := service.NewLocalAuthService(userRepo, tokenRepo, identityRepo, &config.Config{
		JWTSecret:           testJWTSecret,
		JWTAccessExpiryMin:  15,
		JWTRefreshExpiryDay: 7,
	})
	if err != nil {
		panic(err)
	}
	return svc
}

func newPendingLinkToken(t *testing.T, googleSub, email string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   googleSub,
		"email": email,
		"iss":   service.LinkIssuer,
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign pending token: %v", err)
	}
	return tok
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
	_, err := service.NewLocalAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, &config.Config{JWTSecret: ""})
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

// ── Tests: ConfirmAccountLink ─────────────────────────────────────────────────

func TestLocalAuth_ConfirmAccountLink_Success(t *testing.T) {
	hash := mustBcrypt("password123")
	user := &model.User{ID: uuid.New(), Email: "alice@example.com", PasswordHash: &hash}
	identityRepo := &mockUserIdentityRepo{}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, identityRepo)

	_, pair, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if pair == nil || pair.AccessToken == "" {
		t.Fatal("expected token pair")
	}
	if identityRepo.createCalls != 1 || identityRepo.created[0].IsPrimary {
		t.Fatal("expected one non-primary identity row")
	}
}

func TestLocalAuth_ConfirmAccountLink_InvalidToken(t *testing.T) {
	svc := newTestLocalAuthSvc(&mockUserRepo{}, &mockRefreshTokenRepo{})
	_, _, err := svc.ConfirmAccountLink("not-a-jwt", "password")
	if !errors.Is(err, service.ErrInvalidPendingToken) {
		t.Fatalf("expected ErrInvalidPendingToken, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_WrongIssuer(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":   "google-oauth2|xyz",
		"email": "alice@example.com",
		"iss":   service.LocalIssuer,
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))

	svc := newTestLocalAuthSvc(&mockUserRepo{}, &mockRefreshTokenRepo{})
	_, _, err := svc.ConfirmAccountLink(tok, "password")
	if !errors.Is(err, service.ErrInvalidPendingToken) {
		t.Fatalf("expected ErrInvalidPendingToken for wrong issuer, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_ExpiredToken(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":   "google-oauth2|xyz",
		"email": "alice@example.com",
		"iss":   service.LinkIssuer,
		"exp":   time.Now().Add(-1 * time.Minute).Unix(),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))

	svc := newTestLocalAuthSvc(&mockUserRepo{}, &mockRefreshTokenRepo{})
	_, _, err := svc.ConfirmAccountLink(tok, "password")
	if !errors.Is(err, service.ErrInvalidPendingToken) {
		t.Fatalf("expected ErrInvalidPendingToken for expired token, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{})

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "ghost@example.com"), "password")
	if !errors.Is(err, service.ErrInvalidPendingToken) {
		t.Fatalf("expected ErrInvalidPendingToken when user not found, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_NilUserFromRepo(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailUser: nil, findByEmailErr: nil}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{})

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password")
	if !errors.Is(err, service.ErrInvalidPendingToken) {
		t.Fatalf("expected ErrInvalidPendingToken for nil user, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_ZeroUUIDUser(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailUser: &model.User{}} // ID is uuid.Nil
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{})

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password")
	if !errors.Is(err, service.ErrInvalidPendingToken) {
		t.Fatalf("expected ErrInvalidPendingToken for zero-UUID user, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_NoPasswordSet(t *testing.T) {
	user := &model.User{ID: uuid.New(), Email: "alice@example.com"}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{})

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password")
	if !errors.Is(err, service.ErrNoPasswordSet) {
		t.Fatalf("expected ErrNoPasswordSet, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_WrongPassword(t *testing.T) {
	hash := mustBcrypt("correct-password")
	user := &model.User{ID: uuid.New(), Email: "alice@example.com", PasswordHash: &hash}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{})

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "wrong-password")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_RaceLinkedToSameUser(t *testing.T) {
	hash := mustBcrypt("password123")
	user := &model.User{ID: uuid.New(), Email: "alice@example.com", PasswordHash: &hash}
	identityRepo := &mockUserIdentityRepo{
		createErr:             errors.New("duplicate"),
		findBySubjectIdentity: &model.UserIdentity{UserID: user.ID},
	}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, identityRepo)

	result, pair, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password123")
	if err != nil {
		t.Fatalf("expected nil error on race recovery, got %v", err)
	}
	if result.ID != user.ID || pair == nil {
		t.Fatal("expected user and tokens after race recovery")
	}
}

func TestLocalAuth_ConfirmAccountLink_RaceLinkedToDifferentUser(t *testing.T) {
	hash := mustBcrypt("password123")
	user := &model.User{ID: uuid.New(), Email: "alice@example.com", PasswordHash: &hash}
	identityRepo := &mockUserIdentityRepo{
		createErr:             errors.New("duplicate"),
		findBySubjectIdentity: &model.UserIdentity{UserID: uuid.New()},
	}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, identityRepo)

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password123")
	if !errors.Is(err, service.ErrLinkConflict) {
		t.Fatalf("expected ErrLinkConflict, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_InsertFailsLookupFails(t *testing.T) {
	hash := mustBcrypt("password123")
	user := &model.User{ID: uuid.New(), Email: "alice@example.com", PasswordHash: &hash}
	insertErr := errors.New("insert failed")
	identityRepo := &mockUserIdentityRepo{
		createErr:        insertErr,
		findBySubjectErr: errors.New("lookup failed"),
	}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, identityRepo)

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password123")
	if !errors.Is(err, insertErr) {
		t.Fatalf("expected original insert error, got %v", err)
	}
}
