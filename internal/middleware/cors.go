package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS enforces an origin allowlist.
//
// With a non-empty allowlist, only listed origins are reflected back in
// Access-Control-Allow-Origin and credentials are permitted. Echoing the exact
// origin (not "*") is mandatory when credentials are involved: browsers reject
// "*" together with Access-Control-Allow-Credentials: true.
//
// An empty allowlist is a dev-only fallback: it replies "*" WITHOUT credentials
// (the only combination browsers accept for an open CORS policy).
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o = strings.TrimSpace(o); o != "" {
			allowed[o] = struct{}{}
		}
	}
	allowAll := len(allowed) == 0

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if allowAll {
				c.Header("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
			}
			// Origin present but not allowlisted: send no CORS headers so the
			// browser blocks the cross-origin response.
		}

		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
