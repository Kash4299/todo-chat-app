package middleware

import (
	"net/http"
	"os"
	"strings"

	"todo/common"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthenticateMiddleware is a Gin middleware that validates JWT tokens.
// It checks the Authorization header (Bearer <token>) first, then falls back to a cookie.
// On success it stores the claims in the Gin context under "uid", "username", and "email".
func AuthenticateMiddleware(c *gin.Context) {
	tokenString := extractToken(c)
	if tokenString == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, common.UnauthorizedResponse("missing or invalid token"))
		return
	}

	claims, err := parseToken(tokenString)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, common.UnauthorizedResponse("invalid or expired token"))
		return
	}

	// Store claims in context so handlers can access them
	if uid, ok := claims["uid"].(string); ok {
		c.Set("uid", uid)
	}
	if username, ok := claims["username"].(string); ok {
		c.Set("username", username)
	}
	if email, ok := claims["email"].(string); ok {
		c.Set("email", email)
	}

	c.Next()
}

// extractToken tries to get the JWT from the Authorization header,
// falling back to a "Bearer" cookie.
func extractToken(c *gin.Context) string {
	// 1. Try Authorization header: "Bearer <token>"
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	// 2. Fallback: cookie
	if cookie, err := c.Cookie("Bearer"); err == nil && cookie != "" {
		return cookie
	}

	return ""
}

// parseToken validates and parses a JWT token string, returning its claims.
func parseToken(tokenString string) (jwt.MapClaims, error) {
	jwtSecret := os.Getenv("JWT_SECRET")

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}
