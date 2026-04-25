package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

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

func TestExtractBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer abc")

	token := extractBearer(c)
	if token != "abc" {
		t.Fatalf("expected abc, got %s", token)
	}
}

func TestHasRecentStepUp(t *testing.T) {
	claims := jwt.MapClaims{
		"auth_time": float64(time.Now().Add(-2 * time.Minute).Unix()),
		"amr":       []any{"pwd"},
	}
	if !hasRecentStepUp(claims) {
		t.Fatal("expected step-up true")
	}
}

func TestHasRecentStepUpRejectsOldAuthTime(t *testing.T) {
	claims := jwt.MapClaims{
		"auth_time": float64(time.Now().Add(-20 * time.Minute).Unix()),
		"amr":       []any{"mfa"},
	}
	if hasRecentStepUp(claims) {
		t.Fatal("expected step-up false")
	}
}

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

func (m *mockUserServiceForMW) ConfirmAccountLink(auth0ID, email, displayName, avatarURL string, emailVerified, consent, stepUp bool) (*model.User, error) {
	return nil, nil
}

func (m *mockUserServiceForMW) GetByAuth0ID(auth0ID string) (*model.User, error) {
	return nil, nil
}

func (m *mockUserServiceForMW) GetByID(id uuid.UUID) (*model.User, error) {
	return nil, nil
}

func TestAuthMiddleware_AllowsPendingLinkRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	mw := &AuthMiddleware{
		userService: &mockUserServiceForMW{syncErr: service.ErrUserLinkingRequired},
		issuer:      "https://issuer/",
		audience:    "aud",
		jwksByK: map[string]*rsa.PublicKey{
			"kid-1": &priv.PublicKey,
		},
	}

	r := gin.New()
	r.POST("/api/v1/auth/link-identities/confirm", mw.Handle(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":      true,
			"step_up": c.GetBool(StepUpVerifiedContextKey),
		})
	})

	token := buildTestToken(t, priv, "kid-1", "https://issuer/", "aud", map[string]any{
		"sub":            "google-oauth2|abc",
		"email":          "a@example.com",
		"email_verified": true,
		"auth_time":      float64(time.Now().Unix()),
		"amr":            []string{"pwd"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/link-identities/confirm", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_BlocksNonLinkRouteWhenLinkingRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	mw := &AuthMiddleware{
		userService: &mockUserServiceForMW{syncErr: service.ErrUserLinkingRequired},
		issuer:      "https://issuer/",
		audience:    "aud",
		jwksByK: map[string]*rsa.PublicKey{
			"kid-1": &priv.PublicKey,
		},
	}

	r := gin.New()
	r.GET("/api/v1/users/me", mw.Handle(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := buildTestToken(t, priv, "kid-1", "https://issuer/", "aud", map[string]any{
		"sub":            "google-oauth2|abc",
		"email":          "a@example.com",
		"email_verified": true,
		"auth_time":      float64(time.Now().Unix()),
		"amr":            []string{"pwd"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAuthMiddleware_BlocksWhenEmailNotVerified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	mw := &AuthMiddleware{
		userService: &mockUserServiceForMW{syncErr: service.ErrUserEmailNotVerified},
		issuer:      "https://issuer/",
		audience:    "aud",
		jwksByK: map[string]*rsa.PublicKey{
			"kid-1": &priv.PublicKey,
		},
	}

	r := gin.New()
	r.GET("/api/v1/users/me", mw.Handle(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := buildTestToken(t, priv, "kid-1", "https://issuer/", "aud", map[string]any{
		"sub":            "google-oauth2|abc",
		"email":          "a@example.com",
		"email_verified": false,
		"auth_time":      float64(time.Now().Unix()),
		"amr":            []string{"pwd"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

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
	for k, v := range extraClaims {
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestHasRecentStepUpRejectsMissingAmr(t *testing.T) {
	claims := jwt.MapClaims{
		"auth_time": float64(time.Now().Add(-2 * time.Minute).Unix()),
	}
	if hasRecentStepUp(claims) {
		t.Fatal("expected step-up false")
	}
}

func TestExtractBearerMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	token := extractBearer(c)
	if token != "" {
		t.Fatalf("expected empty token, got %s", token)
	}
}

func TestDecodeRSAPublicKeyRejectsBadInput(t *testing.T) {
	_, err := decodeRSAPublicKey("%%%", "%%%")
	if err == nil {
		t.Fatal("expected decode error")
	}
}
