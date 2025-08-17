package http

import (
	"askshop/services/user-service/internal/domain"
	"askshop/services/user-service/internal/service"
	"askshop/services/user-service/pkg/types"
	"askshop/shared/contracts"
	"askshop/shared/env"
	"askshop/shared/response"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Email     string `json:"email"`
	Age       int    `json:"age,omitempty"`
	// Derived / legacy combined name (not part of user-service payload directly)
	Name string `json:"name,omitempty"`
	// Never exposed, placeholder if needed for internal flows
	Password string `json:"-"`
}

type UserHandler struct {
	userService service.UserServiceInterface
}

func NewUserHandler(userService service.UserServiceInterface) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// RegisterRoutes registers the routes for the user handler
func (h *UserHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		users := api.Group("/users")
		{
			users.GET("/:identifier", h.GetUser)
			users.POST("/", h.CreateUser)
		}
	}
}

// mapDomainError maps domain/service errors to HTTP status and standardized codes/messages
func mapDomainError(err error) (int, string, string, interface{}) {
	switch {
	case errors.Is(err, domain.ErrUserAlreadyExists):
		return http.StatusConflict, contracts.CodeUserAlreadyExists, domain.ErrUserAlreadyExists.Error(), nil
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, contracts.CodeInvalidCredentials, domain.ErrInvalidCredentials.Error(), nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound, contracts.CodeUserNotFound, gorm.ErrRecordNotFound.Error(), nil
	default:
		return http.StatusInternalServerError, contracts.CodeInternalServerError, "Internal server error", err.Error()
	}
}

// GetUser handles both ID and email-based user lookup
// Example usage:
// GET /api/users/123e4567-e89b-12d3-a456-426614174000 (UUID)
// GET /api/users/user@example.com (email)
func (h *UserHandler) GetUser(c *gin.Context) {
	identifier := c.Param("identifier")

	if identifier == "" {
		response.Error(c, http.StatusBadRequest, contracts.CodeUserIdentifierRequired, "User identifier (ID or email) is required", nil)
		return
	}

	user, err := h.userService.GetUserByIDOrEmail(c, identifier)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	response.Success(c, http.StatusOK, user, nil, "")
}

// GetUserById handles ID-specific user lookup
func (h *UserHandler) GetUserById(c *gin.Context) {
	userID := c.Param("id")

	if userID == "" {
		response.Error(c, http.StatusBadRequest, contracts.CodeUserIDRequired, "User ID is required", nil)
		return
	}

	user, err := h.userService.GetUserById(c, userID)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	response.Success(c, http.StatusOK, user, nil, "")
}

// GetUserByEmail handles email-specific user lookup
func (h *UserHandler) GetUserByEmail(c *gin.Context) {
	email := c.Query("email")

	if email == "" {
		response.Error(c, http.StatusBadRequest, contracts.CodeEmailRequired, "Email is required", nil)
		return
	}

	user, err := h.userService.GetUserByEmail(c, email)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	response.Success(c, http.StatusOK, user, nil, "")
}

// CreateUser handles user creation
func (h *UserHandler) CreateUser(c *gin.Context) {
	var userRequest types.UserRegistrationRequest
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", err.Error())
		return
	}

	// If Supabase is configured, create the user there first and persist locally using the external id
	supabaseURL := ""
	supabaseKey := ""
	// try to read env via shared env package without importing it here to avoid cycles
	// we expect the service main to call RegisterRoutes with correct configuration; fallback to local create

	// Attempt to create via Supabase if env vars are provided in headers (convention) or skip
	supabaseURL = c.GetHeader("X-SUPABASE-URL")
	supabaseKey = c.GetHeader("X-SUPABASE-KEY")

	if supabaseURL != "" && supabaseKey != "" {
		// Build request to Supabase Admin Users endpoint
		endpoint := strings.TrimRight(supabaseURL, "/") + "/auth/v1/admin/users"

		payload := map[string]interface{}{
			"email":         userRequest.Email,
			"email_confirm": true,
		}
		// include password if provided
		if userRequest.Password != "" {
			payload["password"] = userRequest.Password
		}
		// user metadata
		payload["user_metadata"] = map[string]interface{}{
			"first_name": userRequest.FirstName,
			"last_name":  userRequest.LastName,
		}

		bodyBytes, _ := json.Marshal(payload)
		req, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError, "failed to build supabase request", err.Error())
			return
		}
		req.Header.Set("apikey", supabaseKey)
		req.Header.Set("Authorization", "Bearer "+supabaseKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError, "failed to call supabase", err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			var errBody map[string]interface{}
			_ = json.NewDecoder(resp.Body).Decode(&errBody)
			response.Error(c, http.StatusBadGateway, contracts.CodeInternalServerError, "supabase user creation failed", errBody)
			return
		}

		var supRes map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&supRes); err != nil {
			response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError, "failed to parse supabase response", err.Error())
			return
		}

		externalID := ""
		if idv, ok := supRes["id"]; ok {
			if s, ok := idv.(string); ok {
				externalID = s
			}
		}

		// Persist locally using external id
		user, err := h.userService.RegisterUserWithExternalID(c, userRequest, externalID)
		if err != nil {
			status, code, msg, details := mapDomainError(err)
			response.Error(c, status, code, msg, details)
			return
		}

		response.Success(c, http.StatusCreated, user, nil, "User created successfully")
		return
	}

	// Fallback: create locally
	user, err := h.userService.RegisterUser(c, userRequest)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	uresp := UserResponse{
		ID:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Age:       int(user.Age),
		Name:      user.FirstName + user.LastName,
		Password:  user.Password,
	}
	response.Success(c, http.StatusCreated, uresp, nil, "User created successfully")
}

// Login handles user login. If SUPABASE_URL and SUPABASE_KEY are set in env,
// it will authenticate against Supabase and return the access/refresh tokens.
// Otherwise it will fall back to local password validation and return a local token.
func (h *UserHandler) Login(c *gin.Context) {
	var req types.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", err.Error())
		return
	}

	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")

	if supabaseURL != "" && supabaseKey != "" {
		// Call Supabase token endpoint to sign in
		endpoint := strings.TrimRight(supabaseURL, "/") + "/auth/v1/token?grant_type=password"
		payload := map[string]interface{}{
			"email":    req.Email,
			"password": req.Password,
		}
		bodyBytes, _ := json.Marshal(payload)
		httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError, "failed to build supabase request", err.Error())
			return
		}
		httpReq.Header.Set("apikey", supabaseKey)
		httpReq.Header.Set("Authorization", "Bearer "+supabaseKey)
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			response.Error(c, http.StatusBadGateway, contracts.CodeInternalServerError, "supabase auth request failed", err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			var errBody map[string]interface{}
			_ = json.NewDecoder(resp.Body).Decode(&errBody)
			response.Error(c, http.StatusUnauthorized, contracts.CodeInvalidCredentials, "invalid credentials", errBody)
			return
		}

		var tokenResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError, "failed to parse supabase response", err.Error())
			return
		}

		accessToken := ""
		refreshToken := ""
		if at, ok := tokenResp["access_token"].(string); ok {
			accessToken = at
		}
		if rt, ok := tokenResp["refresh_token"].(string); ok {
			refreshToken = rt
		}

		// Try to extract user info from tokenResp.user
		var supUser map[string]interface{}
		if u, ok := tokenResp["user"].(map[string]interface{}); ok {
			supUser = u
		}

		// Ensure local user exists (create if missing)
		localUser, err := h.userService.GetUserByEmail(c, req.Email)
		if err != nil {
			// If not found, create local user with external id
			externalID := ""
			if supUser != nil {
				if idv, ok := supUser["id"].(string); ok {
					externalID = idv
				}
			}
			regReq := types.UserRegistrationRequest{
				FirstName: "",
				LastName:  "",
				Email:     req.Email,
				Password:  "",
			}
			if supUser != nil {
				if md, ok := supUser["user_metadata"].(map[string]interface{}); ok {
					if fn, ok := md["first_name"].(string); ok {
						regReq.FirstName = fn
					}
					if ln, ok := md["last_name"].(string); ok {
						regReq.LastName = ln
					}
				}
			}
			localUser, _ = h.userService.RegisterUserWithExternalID(c, regReq, externalID)
		}

		// Return local user and tokens
		data := map[string]interface{}{
			"user":          localUser,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		}
		response.Success(c, http.StatusOK, data, nil, "Login successful")
		return
	}

	// Fallback to local authentication
	user, token, err := h.userService.LoginUser(c, req)
	if err != nil {
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	data := map[string]interface{}{
		"user":         user,
		"access_token": token,
	}
	response.Success(c, http.StatusOK, data, nil, "Login successful")
}
