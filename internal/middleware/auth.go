package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const UserIDContextKey = "authedUserID"

type AuthMiddleware struct {
	jwksURL  string
	issuer   string
	audience string
	userSvc  service.IUserService

	mu   sync.Mutex
	jwks keyfunc.Keyfunc
}

func NewAuthMiddleware(userSvc service.IUserService, cfg *config.Config) (*AuthMiddleware, error) {
	if cfg.Auth0Domain == "" {
		return nil, fmt.Errorf("AUTH0_DOMAIN is required")
	}
	if cfg.Auth0Audience == "" {
		return nil, fmt.Errorf("AUTH0_AUDIENCE is required")
	}
	return &AuthMiddleware{
		jwksURL:  fmt.Sprintf("https://%s/.well-known/jwks.json", cfg.Auth0Domain),
		issuer:   fmt.Sprintf("https://%s/", cfg.Auth0Domain),
		audience: cfg.Auth0Audience,
		userSvc:  userSvc,
	}, nil
}

// getJWKS initializes the JWKS client on the first call. If Auth0 is
// unreachable at that moment the error is returned and the client stays nil,
// allowing the next request to retry.
func (m *AuthMiddleware) getJWKS() (keyfunc.Keyfunc, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.jwks == nil {
		k, err := keyfunc.NewDefaultCtx(context.Background(), []string{m.jwksURL})
		if err != nil {
			return nil, err
		}
		m.jwks = k
	}
	return m.jwks, nil
}

func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		jwks, err := m.getJWKS()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unavailable"})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenStr, jwks.Keyfunc,
			jwt.WithIssuer(m.issuer),
			jwt.WithAudience(m.audience),
			jwt.WithValidMethods([]string{"RS256"}),
		)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		sub, _ := claims["sub"].(string)
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing sub claim"})
			return
		}

		user, err := m.userSvc.GetByAuth0ID(sub)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		c.Set(UserIDContextKey, user.ID)
		c.Next()
	}
}
