package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/model"
	emailverificationrepo "github.com/Kash4299/todo-chat-app/internal/repository/emailverification"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ── Mock: refresh token repo ──────────────────────────────────────────────────

type mockRefreshTokenRepo struct {
	saveErr         error
	storedToken     *model.RefreshToken
	findByHashErr   error
	deleteByHashErr error
}

func (m *mockRefreshTokenRepo) Save(token *model.RefreshToken) error { return m.saveErr }
func (m *mockRefreshTokenRepo) FindByHash(hash string) (*model.RefreshToken, error) {
	return m.storedToken, m.findByHashErr
}
func (m *mockRefreshTokenRepo) DeleteByHash(hash string) error           { return m.deleteByHashErr }
func (m *mockRefreshTokenRepo) DeleteAllByUserID(userID uuid.UUID) error { return nil }

// ── Mock: email verification repo ─────────────────────────────────────────────

type mockEmailVerificationRepo struct {
	createErr         error
	findByHashToken   *model.EmailVerificationToken
	findByHashErr     error
	findLatestToken   *model.EmailVerificationToken
	findLatestErr     error
	consumeUserID     uuid.UUID
	consumeErr        error
	consumeCalls      int
	deleteByHashErr   error
	deleteByUserErr   error
	createCalls       int
	deleteByUserCalls int
}

func (m *mockEmailVerificationRepo) Create(token *model.EmailVerificationToken) error {
	m.createCalls++
	return m.createErr
}
func (m *mockEmailVerificationRepo) FindByHash(hash string) (*model.EmailVerificationToken, error) {
	return m.findByHashToken, m.findByHashErr
}
func (m *mockEmailVerificationRepo) FindLatestByUserID(userID uuid.UUID) (*model.EmailVerificationToken, error) {
	return m.findLatestToken, m.findLatestErr
}
func (m *mockEmailVerificationRepo) ConsumeValidByHash(hash string, now time.Time) (uuid.UUID, error) {
	m.consumeCalls++
	return m.consumeUserID, m.consumeErr
}
func (m *mockEmailVerificationRepo) DeleteByHash(hash string) error { return m.deleteByHashErr }
func (m *mockEmailVerificationRepo) DeleteByUserID(id uuid.UUID) error {
	m.deleteByUserCalls++
	return m.deleteByUserErr
}

var _ emailverificationrepo.IEmailVerificationRepository = (*mockEmailVerificationRepo)(nil)

// ── Mock: email service ───────────────────────────────────────────────────────

type mockEmailService struct {
	sendErr   error
	sendCalls int
}

func (m *mockEmailService) SendVerificationEmail(toEmail, rawToken string) error {
	m.sendCalls++
	return m.sendErr
}

var _ service.IEmailService = (*mockEmailService)(nil)

// ── Helpers ───────────────────────────────────────────────────────────────────

const testJWTSecret = "test-secret-at-least-32-characters!!"

func newTestLocalAuthSvc(userRepo *mockUserRepo, tokenRepo *mockRefreshTokenRepo) service.ILocalAuthService {
	return newTestLocalAuthSvcFull(userRepo, tokenRepo, &mockUserIdentityRepo{}, &mockEmailVerificationRepo{}, &mockEmailService{})
}

func newTestLocalAuthSvcFull(
	userRepo *mockUserRepo,
	tokenRepo *mockRefreshTokenRepo,
	identityRepo *mockUserIdentityRepo,
	emailVerRepo *mockEmailVerificationRepo,
	emailSvc *mockEmailService,
) service.ILocalAuthService {
	svc, err := service.NewLocalAuthService(
		userRepo, tokenRepo, identityRepo, emailVerRepo, emailSvc,
		&config.Config{
			JWTSecret:           testJWTSecret,
			JWTAccessExpiryMin:  15,
			JWTRefreshExpiryDay: 7,
		},
	)
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
	_, err := service.NewLocalAuthService(
		&mockUserRepo{}, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{},
		&mockEmailVerificationRepo{}, &mockEmailService{},
		&config.Config{JWTSecret: ""},
	)
	if err == nil {
		t.Fatal("expected error for empty JWT_SECRET")
	}
}

// ── Tests: Register ───────────────────────────────────────────────────────────

func TestLocalAuth_Register_Success(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	user, err := svc.Register("Test@Example.com", "password123", "Test User")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user == nil || user.Email != "test@example.com" {
		t.Fatal("expected user with lowercased email")
	}
	if user.EmailVerified {
		t.Fatal("expected EmailVerified = false after register")
	}
	if user.PasswordHash == nil || *user.PasswordHash == "password123" {
		t.Fatal("expected password to be bcrypt-hashed")
	}
}

func TestLocalAuth_Register_DisplayNameFallsBackToEmail(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	user, err := svc.Register("a@example.com", "password123", "")
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

	_, err := svc.Register("a@example.com", "password123", "A")
	if !errors.Is(err, service.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestLocalAuth_Register_PasswordTooShort(t *testing.T) {
	svc := newTestLocalAuthSvc(&mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}, &mockRefreshTokenRepo{})

	_, err := svc.Register("a@example.com", "short", "A")
	if !errors.Is(err, service.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestLocalAuth_Register_FindByEmailRepoError(t *testing.T) {
	expected := errors.New("db error")
	svc := newTestLocalAuthSvc(&mockUserRepo{findByEmailErr: expected}, &mockRefreshTokenRepo{})

	_, err := svc.Register("a@example.com", "password123", "A")
	if !errors.Is(err, expected) {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestLocalAuth_Register_CreateUserError(t *testing.T) {
	expected := errors.New("create failed")
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound, createErr: expected}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	_, err := svc.Register("a@example.com", "password123", "A")
	if !errors.Is(err, expected) {
		t.Fatalf("expected create error, got %v", err)
	}
}

func TestLocalAuth_Register_EmailSendError(t *testing.T) {
	expected := errors.New("smtp failed")
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}
	emailSvc := &mockEmailService{sendErr: expected}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, &mockEmailVerificationRepo{}, emailSvc)

	_, err := svc.Register("a@example.com", "password123", "A")
	if !errors.Is(err, expected) {
		t.Fatalf("expected smtp error, got %v", err)
	}
	if userRepo.deleteCalls != 1 {
		t.Fatalf("expected rollback delete call, got %d", userRepo.deleteCalls)
	}
}

func TestLocalAuth_Register_EmailSendError_RollbackDeleteFails(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailErr: gorm.ErrRecordNotFound,
		deleteErr:      errors.New("delete failed"),
	}
	emailSvc := &mockEmailService{sendErr: errors.New("smtp failed")}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, &mockEmailVerificationRepo{}, emailSvc)

	_, err := svc.Register("a@example.com", "password123", "A")
	if err == nil {
		t.Fatal("expected error when rollback delete fails")
	}
}

func TestLocalAuth_ResendVerification_SendsForUnverifiedUser(t *testing.T) {
	user := &model.User{ID: uuid.New(), Email: "a@example.com", EmailVerified: false}
	userRepo := &mockUserRepo{findByEmailUser: user}
	emailRepo := &mockEmailVerificationRepo{}
	emailSvc := &mockEmailService{}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, emailRepo, emailSvc)

	err := svc.ResendVerification("a@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if emailRepo.createCalls != 1 {
		t.Fatalf("expected one token create, got %d", emailRepo.createCalls)
	}
	if emailRepo.deleteByUserCalls != 1 {
		t.Fatalf("expected old token cleanup, got %d", emailRepo.deleteByUserCalls)
	}
	if emailSvc.sendCalls != 1 {
		t.Fatalf("expected one email send, got %d", emailSvc.sendCalls)
	}
}

func TestLocalAuth_ResendVerification_RateLimited(t *testing.T) {
	user := &model.User{ID: uuid.New(), Email: "a@example.com", EmailVerified: false}
	userRepo := &mockUserRepo{findByEmailUser: user}
	emailRepo := &mockEmailVerificationRepo{
		findLatestToken: &model.EmailVerificationToken{
			UserID:    user.ID,
			CreatedAt: time.Now().Add(-30 * time.Second),
		},
	}
	emailSvc := &mockEmailService{}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, emailRepo, emailSvc)

	err := svc.ResendVerification("a@example.com")
	if !errors.Is(err, service.ErrVerificationEmailRateLimited) {
		t.Fatalf("expected ErrVerificationEmailRateLimited, got %v", err)
	}
	if emailSvc.sendCalls != 0 {
		t.Fatalf("expected no email send, got %d", emailSvc.sendCalls)
	}
}

func TestLocalAuth_ResendVerification_NoOpWhenVerified(t *testing.T) {
	user := &model.User{ID: uuid.New(), Email: "a@example.com", EmailVerified: true}
	userRepo := &mockUserRepo{findByEmailUser: user}
	emailRepo := &mockEmailVerificationRepo{}
	emailSvc := &mockEmailService{}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, emailRepo, emailSvc)

	err := svc.ResendVerification("a@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if emailRepo.createCalls != 0 {
		t.Fatalf("expected no token create, got %d", emailRepo.createCalls)
	}
	if emailSvc.sendCalls != 0 {
		t.Fatalf("expected no email send, got %d", emailSvc.sendCalls)
	}
}

func TestLocalAuth_ResendVerification_NoOpWhenUserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}
	emailRepo := &mockEmailVerificationRepo{}
	emailSvc := &mockEmailService{}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, emailRepo, emailSvc)

	err := svc.ResendVerification("missing@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if emailRepo.createCalls != 0 {
		t.Fatalf("expected no token create, got %d", emailRepo.createCalls)
	}
	if emailSvc.sendCalls != 0 {
		t.Fatalf("expected no email send, got %d", emailSvc.sendCalls)
	}
}

func TestLocalAuth_ResendVerification_NoOpWhenRepoReturnsNilUser(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailUser: nil}
	emailRepo := &mockEmailVerificationRepo{}
	emailSvc := &mockEmailService{}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, emailRepo, emailSvc)

	err := svc.ResendVerification("a@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if emailSvc.sendCalls != 0 {
		t.Fatalf("expected no email send, got %d", emailSvc.sendCalls)
	}
}

func TestLocalAuth_ResendVerification_UserRepoError(t *testing.T) {
	expected := errors.New("db error")
	userRepo := &mockUserRepo{findByEmailErr: expected}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, &mockEmailVerificationRepo{}, &mockEmailService{})

	err := svc.ResendVerification("a@example.com")
	if !errors.Is(err, expected) {
		t.Fatalf("expected repo error, got %v", err)
	}
}

// ── Tests: Login ──────────────────────────────────────────────────────────────

func TestLocalAuth_Login_Success(t *testing.T) {
	hash := mustBcrypt("password123")
	userRepo := &mockUserRepo{findByEmailUser: &model.User{
		ID: uuid.New(), Email: "a@example.com", PasswordHash: &hash, EmailVerified: true,
	}}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	user, pair, err := svc.Login("a@example.com", "password123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user == nil || pair == nil {
		t.Fatal("expected user and token pair")
	}
}

func TestLocalAuth_Login_EmailNotVerified(t *testing.T) {
	hash := mustBcrypt("password123")
	userRepo := &mockUserRepo{findByEmailUser: &model.User{
		ID: uuid.New(), Email: "a@example.com", PasswordHash: &hash, EmailVerified: false,
	}}
	svc := newTestLocalAuthSvc(userRepo, &mockRefreshTokenRepo{})

	_, _, err := svc.Login("a@example.com", "password123")
	if !errors.Is(err, service.ErrEmailNotVerified) {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}
}

func TestLocalAuth_Login_WrongPassword(t *testing.T) {
	hash := mustBcrypt("correct-password")
	userRepo := &mockUserRepo{findByEmailUser: &model.User{
		ID: uuid.New(), Email: "a@example.com", PasswordHash: &hash, EmailVerified: true,
	}}
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

// ── Tests: VerifyEmail ────────────────────────────────────────────────────────

func TestLocalAuth_VerifyEmail_Success(t *testing.T) {
	userID := uuid.New()
	user := &model.User{ID: userID, Email: "a@example.com", EmailVerified: false}
	emailVerRepo := &mockEmailVerificationRepo{consumeUserID: userID}
	userRepo := &mockUserRepo{findByIDUser: user}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, emailVerRepo, &mockEmailService{})

	resultUser, pair, err := svc.VerifyEmail("raw-token")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !resultUser.EmailVerified {
		t.Fatal("expected EmailVerified = true after verification")
	}
	if pair == nil || pair.AccessToken == "" {
		t.Fatal("expected token pair after verification")
	}
	if userRepo.updateCalls != 1 {
		t.Fatalf("expected one update call, got %d", userRepo.updateCalls)
	}
	if emailVerRepo.consumeCalls != 1 {
		t.Fatalf("expected one consume call, got %d", emailVerRepo.consumeCalls)
	}
}

func TestLocalAuth_VerifyEmail_InvalidToken(t *testing.T) {
	emailVerRepo := &mockEmailVerificationRepo{consumeErr: gorm.ErrRecordNotFound}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{}, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, emailVerRepo, &mockEmailService{})

	_, _, err := svc.VerifyEmail("bad-token")
	if !errors.Is(err, service.ErrInvalidVerificationToken) {
		t.Fatalf("expected ErrInvalidVerificationToken, got %v", err)
	}
}

func TestLocalAuth_VerifyEmail_RepoError(t *testing.T) {
	expected := errors.New("db offline")
	emailVerRepo := &mockEmailVerificationRepo{consumeErr: expected}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{}, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, emailVerRepo, &mockEmailService{})

	_, _, err := svc.VerifyEmail("token")
	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
}

func TestLocalAuth_VerifyEmail_ExpiredToken(t *testing.T) {
	emailVerRepo := &mockEmailVerificationRepo{consumeErr: gorm.ErrRecordNotFound}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{}, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, emailVerRepo, &mockEmailService{})

	_, _, err := svc.VerifyEmail("expired-token")
	if !errors.Is(err, service.ErrInvalidVerificationToken) {
		t.Fatalf("expected ErrInvalidVerificationToken for expired token, got %v", err)
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
	user := &model.User{ID: uuid.New(), Email: "alice@example.com", PasswordHash: &hash, EmailVerified: false}
	userRepo := &mockUserRepo{findByEmailUser: user}
	identityRepo := &mockUserIdentityRepo{}
	svc := newTestLocalAuthSvcFull(
		userRepo, &mockRefreshTokenRepo{},
		identityRepo, &mockEmailVerificationRepo{}, &mockEmailService{},
	)

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
	if !user.EmailVerified {
		t.Fatal("expected account link to verify local email")
	}
	if userRepo.updateCalls != 1 {
		t.Fatalf("expected one user update, got %d", userRepo.updateCalls)
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
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, &mockEmailVerificationRepo{}, &mockEmailService{})

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "ghost@example.com"), "password")
	if !errors.Is(err, service.ErrInvalidPendingToken) {
		t.Fatalf("expected ErrInvalidPendingToken when user not found, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_NilUserFromRepo(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailUser: nil, findByEmailErr: nil}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, &mockEmailVerificationRepo{}, &mockEmailService{})

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password")
	if !errors.Is(err, service.ErrInvalidPendingToken) {
		t.Fatalf("expected ErrInvalidPendingToken for nil user, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_ZeroUUIDUser(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailUser: &model.User{}}
	svc := newTestLocalAuthSvcFull(userRepo, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, &mockEmailVerificationRepo{}, &mockEmailService{})

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password")
	if !errors.Is(err, service.ErrInvalidPendingToken) {
		t.Fatalf("expected ErrInvalidPendingToken for zero-UUID user, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_NoPasswordSet(t *testing.T) {
	user := &model.User{ID: uuid.New(), Email: "alice@example.com"}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, &mockEmailVerificationRepo{}, &mockEmailService{})

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password")
	if !errors.Is(err, service.ErrNoPasswordSet) {
		t.Fatalf("expected ErrNoPasswordSet, got %v", err)
	}
}

func TestLocalAuth_ConfirmAccountLink_WrongPassword(t *testing.T) {
	hash := mustBcrypt("correct-password")
	user := &model.User{ID: uuid.New(), Email: "alice@example.com", PasswordHash: &hash}
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, &mockUserIdentityRepo{}, &mockEmailVerificationRepo{}, &mockEmailService{})

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
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, identityRepo, &mockEmailVerificationRepo{}, &mockEmailService{})

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
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, identityRepo, &mockEmailVerificationRepo{}, &mockEmailService{})

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
	svc := newTestLocalAuthSvcFull(&mockUserRepo{findByEmailUser: user}, &mockRefreshTokenRepo{}, identityRepo, &mockEmailVerificationRepo{}, &mockEmailService{})

	_, _, err := svc.ConfirmAccountLink(newPendingLinkToken(t, "google-oauth2|xyz", "alice@example.com"), "password123")
	if !errors.Is(err, insertErr) {
		t.Fatalf("expected original insert error, got %v", err)
	}
}
