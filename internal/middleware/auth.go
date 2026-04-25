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
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const UserIDContextKey = "authedUserID"
const Auth0IDContextKey = "authedAuth0ID"
const EmailContextKey = "authedEmail"
const DisplayNameContextKey = "authedDisplayName"
const AvatarURLContextKey = "authedAvatarURL"
const EmailVerifiedContextKey = "authedEmailVerified"
const StepUpVerifiedContextKey = "authedStepUpVerified"

type AuthMiddleware struct {
	userService service.IUserService
	issuer      string
	audience    string
	jwksURL     string
	httpClient  *http.Client

	mu      sync.RWMutex
	jwksByK map[string]*rsa.PublicKey
}

func NewAuthMiddleware(cfg *config.Config, userService service.IUserService) (*AuthMiddleware, error) {
	if strings.TrimSpace(cfg.Auth0Domain) == "" {
		return nil, fmt.Errorf("AUTH0_DOMAIN is required")
	}
	if strings.TrimSpace(cfg.Auth0Audience) == "" {
		return nil, fmt.Errorf("AUTH0_AUDIENCE is required")
	}

	domain, err := normalizeAuth0Domain(cfg.Auth0Domain)
	if err != nil {
		return nil, err
	}

	return &AuthMiddleware{
		userService: userService,
		issuer:      "https://" + domain + "/",
		audience:    cfg.Auth0Audience,
		jwksURL:     "https://" + domain + "/.well-known/jwks.json",
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		jwksByK:     make(map[string]*rsa.PublicKey),
	}, nil
}

func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractBearer(c)
		if tokenStr == "" {
			// Also accept token from query param for WebSocket upgrades
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
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer(m.issuer),
			jwt.WithAudience(m.audience),
		)

		if err != nil || !token.Valid {
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
		stepUpVerified := hasRecentStepUp(claims)

		user, err := m.userService.SyncAuth0User(sub, email, displayName, avatarURL, emailVerified)
		if errors.Is(err, service.ErrUserLinkingRequired) {
			c.Set(Auth0IDContextKey, sub)
			c.Set(EmailContextKey, email)
			c.Set(DisplayNameContextKey, displayName)
			c.Set(AvatarURLContextKey, avatarURL)
			c.Set(EmailVerifiedContextKey, emailVerified)
			c.Set(StepUpVerifiedContextKey, stepUpVerified)
			if c.FullPath() == "/api/v1/auth/link-identities/confirm" {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "account linking required",
				"code":  "ACCOUNT_LINKING_REQUIRED",
			})
			return
		}
		if errors.Is(err, service.ErrUserEmailNotVerified) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "email is not verified",
				"code":  "EMAIL_NOT_VERIFIED",
			})
			return
		}
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user identity not recognized"})
			return
		}
		if user.ID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user identity"})
			return
		}

		c.Set(UserIDContextKey, user.ID)
		c.Set(Auth0IDContextKey, sub)
		c.Next()
	}
}

func hasRecentStepUp(claims jwt.MapClaims) bool {
	authTime := int64(0)
	switch v := claims["auth_time"].(type) {
	case float64:
		authTime = int64(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			authTime = parsed
		}
	}
	if authTime <= 0 {
		return false
	}
	if time.Since(time.Unix(authTime, 0)) > 5*time.Minute {
		return false
	}

	amrRaw, ok := claims["amr"]
	if !ok {
		return false
	}
	amrList, ok := amrRaw.([]any)
	if !ok {
		return false
	}
	for _, factor := range amrList {
		val, _ := factor.(string)
		if val == "mfa" || val == "pwd" {
			return true
		}
	}
	return false
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
	if strings.TrimSpace(kid) == "" {
		return nil, fmt.Errorf("token is missing kid")
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
