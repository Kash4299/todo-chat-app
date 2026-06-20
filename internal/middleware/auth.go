package middleware

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const UserIDContextKey = "authedUserID"
const Auth0IDContextKey = "authedAuth0ID"

type AuthMiddleware struct {
	userService    service.IUserService
	issuer         string
	audience       string
	jwksURL        string
	localJWTSecret string
	httpClient     *http.Client

	auth0Enabled bool

	mu      sync.RWMutex
	jwksByK map[string]*rsa.PublicKey
}

func NewAuthMiddleware(cfg *config.Config, userService service.IUserService) (*AuthMiddleware, error) {
	m := &AuthMiddleware{
		userService:    userService,
		localJWTSecret: cfg.JWTSecret,
		httpClient:     &http.Client{Timeout: 5 * time.Second},
		jwksByK:        make(map[string]*rsa.PublicKey),
	}

	// Auth0 is being removed (T80). It is now optional: the Auth0 verification
	// path is enabled only when its config is fully present. Without it, the
	// middleware trusts only locally-issued HS256 tokens (iss == LocalIssuer).
	domainSet := strings.TrimSpace(cfg.Auth0Domain) != ""
	audienceSet := strings.TrimSpace(cfg.Auth0Audience) != ""
	if domainSet != audienceSet {
		return nil, fmt.Errorf("AUTH0_DOMAIN and AUTH0_AUDIENCE must be set together")
	}
	if domainSet {
		domain, err := normalizeAuth0Domain(cfg.Auth0Domain)
		if err != nil {
			return nil, err
		}
		m.auth0Enabled = true
		m.issuer = "https://" + domain + "/"
		m.audience = cfg.Auth0Audience
		m.jwksURL = "https://" + domain + "/.well-known/jwks.json"
	}

	return m, nil
}

func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractBearer(c)
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(
			tokenStr,
			claims,
			m.keyFunc,
			jwt.WithValidMethods([]string{"RS256", "HS256"}),
		)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		kid, _ := token.Header["kid"].(string)

		if kid != "" {
			m.handleAuth0Token(c, claims)
		} else {
			m.handleLocalToken(c, claims)
		}
	}
}

func (m *AuthMiddleware) handleAuth0Token(c *gin.Context, claims jwt.MapClaims) {
	iss, _ := claims["iss"].(string)
	if iss != m.issuer {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if !hasAudience(claims, m.audience) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	sub, _ := claims["sub"].(string)
	if strings.TrimSpace(sub) == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	email, _ := claims["email"].(string)
	displayName, _ := claims["name"].(string)
	avatarURL, _ := claims["picture"].(string)
	emailVerified, _ := claims["email_verified"].(bool)

	user, err := m.userService.SyncAuth0User(sub, email, displayName, avatarURL, emailVerified)
	if errors.Is(err, constants.ErrUserEmailNotVerified) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "email is not verified",
			"code":  "EMAIL_NOT_VERIFIED",
		})
		return
	}
	var linkRequired *service.LinkRequiredError
	if errors.As(err, &linkRequired) {
		pendingToken, tokenErr := m.issuePendingLinkToken(linkRequired.GoogleSub, linkRequired.Email)
		if tokenErr != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"code":          "ACCOUNT_LINK_REQUIRED",
			"pending_token": pendingToken,
		})
		return
	}
	if err != nil || user == nil || user.ID == uuid.Nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user identity not recognized"})
		return
	}

	c.Set(UserIDContextKey, user.ID)
	c.Set(Auth0IDContextKey, sub)
	c.Next()
}

func (m *AuthMiddleware) handleLocalToken(c *gin.Context, claims jwt.MapClaims) {
	iss, _ := claims["iss"].(string)
	if iss != service.LocalIssuer {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	sub, _ := claims["sub"].(string)
	userID, err := uuid.Parse(sub)
	if err != nil || userID == uuid.Nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	c.Set(UserIDContextKey, userID)
	c.Next()
}

func extractBearer(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if after, ok := strings.CutPrefix(header, "Bearer "); ok {
		return after
	}
	return ""
}

func (m *AuthMiddleware) keyFunc(token *jwt.Token) (any, error) {
	kid, _ := token.Header["kid"].(string)

	if kid != "" {
		if !m.auth0Enabled {
			return nil, fmt.Errorf("auth0 verification disabled")
		}
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("expected RS256 for Auth0 token")
		}
		if key := m.lookupKey(kid); key != nil {
			return key, nil
		}
		if err := m.refreshJWKS(); err != nil {
			return nil, err
		}
		if key := m.lookupKey(kid); key != nil {
			return key, nil
		}
		return nil, fmt.Errorf("no public key found for kid")
	}

	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("expected HS256 for local token")
	}
	if m.localJWTSecret == "" {
		return nil, fmt.Errorf("local JWT not configured")
	}
	return []byte(m.localJWTSecret), nil
}

func (m *AuthMiddleware) issuePendingLinkToken(googleSub, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   googleSub,
		"email": email,
		"iss":   service.LinkIssuer,
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(m.localJWTSecret))
}

func (m *AuthMiddleware) lookupKey(kid string) *rsa.PublicKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.jwksByK[kid]
}

func (m *AuthMiddleware) refreshJWKS() error {
	req, err := http.NewRequest(http.MethodGet, m.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	var parsed jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return err
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, key := range parsed.Keys {
		if key.KTY != "RSA" || key.Kid == "" || key.N == "" || key.E == "" {
			continue
		}
		pub, err := decodeRSAPublicKey(key.N, key.E)
		if err != nil {
			continue
		}
		keys[key.Kid] = pub
	}
	if len(keys) == 0 {
		return fmt.Errorf("no rsa keys found in jwks")
	}

	m.mu.Lock()
	m.jwksByK = keys
	m.mu.Unlock()
	return nil
}

func normalizeAuth0Domain(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("AUTH0_DOMAIN is required")
	}
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		trimmed = "https://" + trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid AUTH0_DOMAIN: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid AUTH0_DOMAIN: missing host")
	}
	return strings.TrimSuffix(u.Host, "/"), nil
}

func hasAudience(claims jwt.MapClaims, audience string) bool {
	aud, ok := claims["aud"]
	if !ok {
		return false
	}
	switch v := aud.(type) {
	case string:
		return v == audience
	case []interface{}:
		for _, a := range v {
			if s, ok := a.(string); ok && s == audience {
				return true
			}
		}
	}
	return false
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	KTY string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func decodeRSAPublicKey(nRaw, eRaw string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nRaw)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eRaw)
	if err != nil {
		return nil, err
	}
	exponent := 0
	for _, b := range eBytes {
		exponent = exponent<<8 + int(b)
	}
	if exponent <= 0 {
		return nil, fmt.Errorf("invalid rsa exponent")
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: exponent,
	}, nil
}
