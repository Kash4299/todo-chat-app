package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ── Mock ──────────────────────────────────────────────────────────────────────

type mockUserServiceForMW struct {
	syncErr error
	user    *model.User
}

func (m *mockUserServiceForMW) SyncAuth0User(auth0ID, email, displayName, avatarURL string, emailVerified bool) (*model.User, error) {
	if m.syncErr != nil {
		return nil, m.syncErr
	}
	if m.user != nil {
		return m.user, nil
	}
	return &model.User{ID: uuid.New()}, nil
}

func TestAuthMiddleware_Auth0_LinkRequired_Returns409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	secret := "test-secret-32-chars-minimum!!!!"

	mw := &AuthMiddleware{
		userService:    &mockUserServiceForMW{syncErr: &service.LinkRequiredError{GoogleSub: "google-oauth2|xyz", Email: "alice@example.com"}},
		issuer:         "https://issuer/",
		audience:       "aud",
		jwksByK:        map[string]*rsa.PublicKey{"kid-1": &priv.PublicKey},
		localJWTSecret: secret,
	}

	r := gin.New()
	r.GET("/me", mw.Handle(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := buildTestToken(t, priv, "kid-1", "https://issuer/", "aud", map[string]any{
		"sub":            "google-oauth2|xyz",
		"email":          "alice@example.com",
		"email_verified": true,
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "ACCOUNT_LINK_REQUIRED") {
		t.Fatalf("expected ACCOUNT_LINK_REQUIRED in body, got: %s", body)
	}
	if !strings.Contains(body, "pending_token") {
		t.Fatalf("expected pending_token in body, got: %s", body)
	}
}

func (m *mockUserServiceForMW) GetByAuth0ID(auth0ID string) (*model.User, error) { return nil, nil }
func (m *mockUserServiceForMW) GetByID(id uuid.UUID) (*model.User, error)        { return nil, nil }

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildTestToken(
	t *testing.T,
	privateKey *rsa.PrivateKey,
	kid, issuer, audience string,
	extraClaims map[string]any,
) string {
	t.Helper()

	claims := jwt.MapClaims{
		"iss": issuer,
		"aud": audience,
		"exp": time.Now().Add(10 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	}
	maps.Copy(claims, extraClaims)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func buildLocalToken(t *testing.T, secret string, userID uuid.UUID, issuer string, expiry time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"iss": issuer,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(expiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign local token: %v", err)
	}
	return signed
}

// ── Tests: normalizeAuth0Domain ───────────────────────────────────────────────

func TestNormalizeAuth0Domain(t *testing.T) {
	domain, err := normalizeAuth0Domain("my-tenant.us.auth0.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if domain != "my-tenant.us.auth0.com" {
		t.Fatalf("unexpected domain: %s", domain)
	}
}

func TestNormalizeAuth0DomainRejectsInvalid(t *testing.T) {
	_, err := normalizeAuth0Domain("http://")
	if err == nil {
		t.Fatal("expected error for invalid domain")
	}
}

// ── Tests: decodeRSAPublicKey ─────────────────────────────────────────────────

func TestDecodeRSAPublicKey(t *testing.T) {
	n := base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3, 4})
	e := base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}) // 65537

	key, err := decodeRSAPublicKey(n, e)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if key.E != 65537 {
		t.Fatalf("expected exponent 65537, got %d", key.E)
	}
}

func TestDecodeRSAPublicKeyRejectsBadInput(t *testing.T) {
	_, err := decodeRSAPublicKey("%%%", "%%%")
	if err == nil {
		t.Fatal("expected decode error")
	}
}

// ── Tests: extractBearer ──────────────────────────────────────────────────────

func TestExtractBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer abc")

	if token := extractBearer(c); token != "abc" {
		t.Fatalf("expected abc, got %s", token)
	}
}

func TestExtractBearerMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	if token := extractBearer(c); token != "" {
		t.Fatalf("expected empty token, got %s", token)
	}
}

// ── Tests: Auth0 RS256 path ───────────────────────────────────────────────────

func TestAuthMiddleware_BlocksWhenEmailNotVerified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	mw := &AuthMiddleware{
		userService: &mockUserServiceForMW{syncErr: constants.ErrUserEmailNotVerified},
		issuer:      "https://issuer/",
		audience:    "aud",
		jwksByK:     map[string]*rsa.PublicKey{"kid-1": &priv.PublicKey},
	}

	r := gin.New()
	r.GET("/api/v1/users/me", mw.Handle(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := buildTestToken(t, priv, "kid-1", "https://issuer/", "aud", map[string]any{
		"sub":            "google-oauth2|abc",
		"email":          "a@example.com",
		"email_verified": false,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAuthMiddleware_Auth0_RejectsWrongIssuer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	mw := &AuthMiddleware{
		userService: &mockUserServiceForMW{},
		issuer:      "https://issuer/",
		audience:    "aud",
		jwksByK:     map[string]*rsa.PublicKey{"kid-1": &priv.PublicKey},
	}

	r := gin.New()
	r.GET("/me", mw.Handle(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := buildTestToken(t, priv, "kid-1", "https://wrong-issuer/", "aud", map[string]any{
		"sub": "auth0|abc",
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ── Tests: local HS256 path ───────────────────────────────────────────────────

func TestAuthMiddleware_LocalToken_ValidHS256(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret-32-chars-minimum!!!!"
	userID := uuid.New()

	mw := &AuthMiddleware{
		localJWTSecret: secret,
		jwksByK:        make(map[string]*rsa.PublicKey),
	}

	r := gin.New()
	r.GET("/me", mw.Handle(), func(c *gin.Context) {
		id, _ := c.Get(UserIDContextKey)
		c.JSON(http.StatusOK, gin.H{"id": id.(uuid.UUID).String()})
	})

	tokenStr := buildLocalToken(t, secret, userID, service.LocalIssuer, 15*time.Minute)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_LocalToken_WrongSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	mw := &AuthMiddleware{
		localJWTSecret: "correct-secret-32-chars-minimum!!",
		jwksByK:        make(map[string]*rsa.PublicKey),
	}

	r := gin.New()
	r.GET("/me", mw.Handle(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	tokenStr := buildLocalToken(t, "wrong-secret-32-chars-minimum!!!!", userID, service.LocalIssuer, 15*time.Minute)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_LocalToken_WrongIssuer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret-32-chars-minimum!!!!"
	userID := uuid.New()

	mw := &AuthMiddleware{
		localJWTSecret: secret,
		jwksByK:        make(map[string]*rsa.PublicKey),
	}

	r := gin.New()
	r.GET("/me", mw.Handle(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	tokenStr := buildLocalToken(t, secret, userID, "wrong-issuer", 15*time.Minute)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
