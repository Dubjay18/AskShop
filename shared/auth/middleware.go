package auth

import (
	"askshop/shared/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Context keys
type contextKey string

const ClaimsContextKey = "token_claims"

// AuthMiddleware returns a Gin middleware that validates JWT access tokens using the provided manager.
// On success, it stores the *Claims in the Gin context under ClaimsContextKey
func AuthMiddleware(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "AUTH_HEADER_REQUIRED", "Authorization header required", nil)
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, http.StatusUnauthorized, "AUTH_INVALID_HEADER", "Invalid authorization header format", nil)
			c.Abort()
			return
		}
		claims, err := m.ParseAccess(parts[1])
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "AUTH_INVALID_TOKEN", "Invalid or expired token", err.Error())
			c.Abort()
			return
		}
		c.Set(ClaimsContextKey, claims)
		c.Next()
	}
}

// OptionalAuthMiddleware allows requests without tokens but parses token if present
func OptionalAuthMiddleware(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}
		if claims, err := m.ParseAccess(parts[1]); err == nil {
			c.Set(ClaimsContextKey, claims)
		}
		c.Next()
	}
}

// GetClaims helper to fetch claims from context
func GetClaims(c *gin.Context) (*Claims, bool) {
	v, ok := c.Get(ClaimsContextKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*Claims)
	return claims, ok
}
