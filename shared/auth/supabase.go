package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"askshop/shared/response"

	"github.com/gin-gonic/gin"
)

// supabaseUser models the minimal subset of the Supabase /auth/v1/user response
type supabaseUser struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	Role         string                 `json:"role,omitempty"`
	UserMetadata map[string]interface{} `json:"user_metadata,omitempty"`
}

// SupabaseAuthMiddleware validates Supabase JWTs by calling Supabase /auth/v1/user.
// On success it maps the returned user into the project's Claims type and stores
// it in the Gin context under ClaimsContextKey.
func SupabaseAuthMiddleware(supabaseURL, supabaseKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if supabaseURL == "" || supabaseKey == "" {
			response.Error(c, http.StatusInternalServerError, "AUTH_CONFIG_MISSING", "Supabase URL or key not configured", nil)
			c.Abort()
			return
		}

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

		// Build request to Supabase user endpoint
		endpoint := strings.TrimRight(supabaseURL, "/") + "/auth/v1/user"
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "AUTH_REQUEST_FAILED", "failed to build validation request", err.Error())
			c.Abort()
			return
		}
		// Forward the bearer token and include apikey header required by Supabase
		req.Header.Set("Authorization", "Bearer "+parts[1])
		req.Header.Set("apikey", supabaseKey)
		req.Header.Set("Accept", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "AUTH_INVALID_TOKEN", "failed to validate token with supabase", err.Error())
			c.Abort()
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// Try to surface a useful message, but don't leak provider internals
			response.Error(c, http.StatusUnauthorized, "AUTH_INVALID_TOKEN", "invalid or expired token", nil)
			c.Abort()
			return
		}

		var su supabaseUser
		if err := json.NewDecoder(resp.Body).Decode(&su); err != nil {
			response.Error(c, http.StatusInternalServerError, "AUTH_PARSE_FAILED", "failed to parse supabase response", err.Error())
			c.Abort()
			return
		}

		// Map supabase user to local Claims shape
		claims := &Claims{
			UserID: su.ID,
			Email:  su.Email,
		}

		// Try to extract first name from user_metadata (common field names)
		if su.UserMetadata != nil {
			if v, ok := su.UserMetadata["first_name"]; ok {
				if s, ok := v.(string); ok {
					claims.FirstName = s
				}
			} else if v, ok := su.UserMetadata["name"]; ok {
				if s, ok := v.(string); ok {
					claims.FirstName = s
				}
			}

			// roles can be provided in metadata as array or comma-separated string
			if v, ok := su.UserMetadata["roles"]; ok {
				switch t := v.(type) {
				case []interface{}:
					for _, it := range t {
						if s, ok := it.(string); ok {
							claims.Roles = append(claims.Roles, s)
						}
					}
				case string:
					// split on commas, trim spaces
					for _, r := range strings.Split(t, ",") {
						r = strings.TrimSpace(r)
						if r != "" {
							claims.Roles = append(claims.Roles, r)
						}
					}
				}
			}
		}

		// fallback: use supabase role if provided
		if len(claims.Roles) == 0 && su.Role != "" {
			claims.Roles = []string{su.Role}
		}

		c.Set(ClaimsContextKey, claims)
		c.Next()
	}
}

// OptionalSupabaseAuthMiddleware behaves like SupabaseAuthMiddleware but allows
// requests without Authorization header. If a valid token is present it will be
// parsed and claims attached to the context, otherwise the request continues.
func OptionalSupabaseAuthMiddleware(supabaseURL, supabaseKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}
		// Reuse SupabaseAuthMiddleware behaviour by calling it inline.
		handler := SupabaseAuthMiddleware(supabaseURL, supabaseKey)
		handler(c)
		if c.IsAborted() {
			return
		}
		c.Next()
	}
}
