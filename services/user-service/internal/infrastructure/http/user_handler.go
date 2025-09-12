package http

import (
	"askshop/services/user-service/internal/domain"
	"askshop/services/user-service/internal/service"
	"askshop/services/user-service/pkg/types"
	supabaseAuth "askshop/shared/auth"
	"askshop/shared/contracts"
	"askshop/shared/env"
	"askshop/shared/logger"
	"askshop/shared/response"
	"errors"
	"net/http"
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
	logger      *logger.Logger
}

func NewUserHandler(userService service.UserServiceInterface) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger.Default("user-handler"),
	}
}

// AuthDiagnostics provides diagnostic information about authentication and Supabase
func (h *UserHandler) AuthDiagnostics(c *gin.Context) {
	ctxLogger := logger.FromGin(c)
	ctxLogger.Info("Auth diagnostics requested")

	// Get environment variables
	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")
	currentEnv := env.GetString("ENV", "development")

	// Create diagnostic result
	diagnostics := map[string]interface{}{
		"environment": currentEnv,
		"supabase": map[string]interface{}{
			"url_configured": supabaseURL != "",
			"key_configured": supabaseKey != "",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// If Supabase is configured, run connection test
	if supabaseURL != "" && supabaseKey != "" {
		supaClient := supabaseAuth.NewSupabaseClient()

		if supaClient != nil && supaClient.IsAvailable() {
			// Get diagnostic info
			diagnostics["supabase"].(map[string]interface{})["client_initialized"] = true
			diagnostics["supabase"].(map[string]interface{})["diagnostics"] = supaClient.GetDiagnostics()
			diagnostics["supabase"].(map[string]interface{})["connection_test"] = supaClient.TestConnection()
		} else {
			diagnostics["supabase"].(map[string]interface{})["client_initialized"] = false
			diagnostics["supabase"].(map[string]interface{})["error"] = "Supabase client initialization failed"
		}
	}

	// Return diagnostic information
	response.Success(c, http.StatusOK, diagnostics, nil, "Authentication diagnostics")
}

// TestLogin provides a direct test of login functionality
func (h *UserHandler) TestLogin(c *gin.Context) {
	ctxLogger := logger.FromGin(c)
	ctxLogger.Info("Test login requested")

	// Get credentials from request body
	var req types.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctxLogger.Errorf("Invalid test login request: %v", err)
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", err.Error())
		return
	}

	// Create result structure
	result := map[string]interface{}{
		"timestamp":       time.Now().Format(time.RFC3339),
		"email":           req.Email,
		"password_length": len(req.Password),
		"tests":           map[string]interface{}{},
	}

	// Try direct Supabase login
	ctxLogger.Info("Testing direct Supabase login")
	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")

	if supabaseURL != "" && supabaseKey != "" {
		supaClient := supabaseAuth.NewSupabaseClient()

		if supaClient != nil && supaClient.IsAvailable() {
			result["tests"].(map[string]interface{})["supabase_client_initialized"] = true

			// Try login
			ctxLogger.Infof("Testing Supabase login with email: %s", req.Email)
			loginResp, err := supaClient.SignIn(c.Request.Context(), req.Email, req.Password)

			if err != nil {
				result["tests"].(map[string]interface{})["supabase_login"] = map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}
				ctxLogger.Errorf("Supabase login test failed: %v", err)
			} else {
				// Success - but don't include tokens in response
				result["tests"].(map[string]interface{})["supabase_login"] = map[string]interface{}{
					"success":             true,
					"user_id":             loginResp["user"] != nil,
					"access_token_length": len(loginResp["access_token"].(string)),
				}
				ctxLogger.Info("Supabase login test successful")
			}
		} else {
			result["tests"].(map[string]interface{})["supabase_client_initialized"] = false
			result["tests"].(map[string]interface{})["error"] = "Supabase client initialization failed"
			ctxLogger.Error("Supabase client initialization failed")
		}
	}

	// Try local login
	ctxLogger.Info("Testing local login")
	user, token, err := h.userService.LoginUser(c, req)

	if err != nil {
		result["tests"].(map[string]interface{})["local_login"] = map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		ctxLogger.Errorf("Local login test failed: %v", err)
	} else {
		result["tests"].(map[string]interface{})["local_login"] = map[string]interface{}{
			"success":      true,
			"user_id":      user.ID.String(),
			"token_length": len(token),
		}
		ctxLogger.Info("Local login test successful")
	}

	response.Success(c, http.StatusOK, result, nil, "Login test completed")
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

		// Authentication routes
		auth := api.Group("/auth")
		{
			// Debug/diagnostic endpoint - only in development
			if env.GetString("ENV", "development") != "production" {
				auth.GET("/diagnostics", h.AuthDiagnostics)
				auth.POST("/test-login", h.TestLogin)
			}
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
	ctxLogger := logger.FromGin(c)
	identifier := c.Param("identifier")

	ctxLogger.Infof("GetUser request received for identifier: %s", identifier)

	if identifier == "" {
		ctxLogger.Warn("GetUser request missing identifier")
		response.Error(c, http.StatusBadRequest, contracts.CodeUserIdentifierRequired, "User identifier (ID or email) is required", nil)
		return
	}

	user, err := h.userService.GetUserByIDOrEmail(c, identifier)
	if err != nil {
		ctxLogger.Errorf("Error retrieving user by identifier %s: %v", identifier, err)
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	ctxLogger.Infof("Successfully retrieved user with identifier: %s", identifier)

	// Convert user domain model to response format with proper ID as string
	userResp := UserResponse{
		ID:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Age:       int(user.Age),
		Name:      user.FirstName + " " + user.LastName,
	}

	response.Success(c, http.StatusOK, userResp, nil, "")
}

// GetUserById handles ID-specific user lookup
func (h *UserHandler) GetUserById(c *gin.Context) {
	ctxLogger := logger.FromGin(c)
	userID := c.Param("id")

	ctxLogger.Infof("GetUserById request received for ID: %s", userID)

	if userID == "" {
		ctxLogger.Warn("GetUserById request missing ID parameter")
		response.Error(c, http.StatusBadRequest, contracts.CodeUserIDRequired, "User ID is required", nil)
		return
	}

	user, err := h.userService.GetUserById(c, userID)
	if err != nil {
		ctxLogger.Errorf("Error retrieving user by ID %s: %v", userID, err)
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	ctxLogger.Infof("Successfully retrieved user with ID: %s", userID)

	// Convert user domain model to response format with proper ID as string
	userResp := UserResponse{
		ID:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Age:       int(user.Age),
		Name:      user.FirstName + " " + user.LastName,
	}

	response.Success(c, http.StatusOK, userResp, nil, "")
}

// GetUserByEmail handles email-specific user lookup
func (h *UserHandler) GetUserByEmail(c *gin.Context) {
	ctxLogger := logger.FromGin(c)
	email := c.Query("email")

	ctxLogger.Infof("GetUserByEmail request received for email: %s", email)

	if email == "" {
		ctxLogger.Warn("GetUserByEmail request missing email parameter")
		response.Error(c, http.StatusBadRequest, contracts.CodeEmailRequired, "Email is required", nil)
		return
	}

	user, err := h.userService.GetUserByEmail(c, email)
	if err != nil {
		ctxLogger.Errorf("Error retrieving user by email %s: %v", email, err)
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	ctxLogger.Infof("Successfully retrieved user with email: %s", email)

	// Convert user domain model to response format with proper ID as string
	userResp := UserResponse{
		ID:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Age:       int(user.Age),
		Name:      user.FirstName + " " + user.LastName,
	}

	response.Success(c, http.StatusOK, userResp, nil, "")
}

// CreateUser handles user creation
func (h *UserHandler) CreateUser(c *gin.Context) {
	ctxLogger := logger.FromGin(c)
	ctxLogger.Info("CreateUser request received")

	var userRequest types.UserRegistrationRequest
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		ctxLogger.Errorf("Invalid request body: %v", err)
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", err.Error())
		return
	}

	ctxLogger.Debugf("Creating user with email: %s", userRequest.Email)

	// If Supabase is configured, create the user there first and persist locally using the external id
	supabaseURL := ""
	supabaseKey := ""
	// try to read env via shared env package without importing it here to avoid cycles
	// we expect the service main to call RegisterRoutes with correct configuration; fallback to local create

	// Attempt to create via Supabase if env vars are provided in headers (convention) or skip
	supabaseURL = env.GetString("SUPABASE_URL", "")
	supabaseKey = env.GetString("SUPABASE_KEY", "")

	if supabaseURL != "" && supabaseKey != "" {
		ctxLogger.Debug("Using Supabase for user creation")
		// Use the new Supabase client
		supaClient := supabaseAuth.NewSupabaseClient()

		if !supaClient.IsAvailable() {
			ctxLogger.Error("Supabase client initialization failed")
			response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError, "Supabase client initialization failed", nil)
			return
		}

		// User metadata
		userData := map[string]interface{}{
			"first_name": userRequest.FirstName,
			"last_name":  userRequest.LastName,
		}

		// Create the user in Supabase
		ctxLogger.Debug("Creating user in Supabase")
		supRes, err := supaClient.SignUp(c.Request.Context(), userRequest.Email, userRequest.Password, userData)
		if err != nil {
			ctxLogger.Errorf("Supabase user creation failed: %v", err)
			response.Error(c, http.StatusBadGateway, contracts.CodeInternalServerError, "Supabase user creation failed", err.Error())
			return
		}
		ctxLogger.Debug("Successfully created user in Supabase")

		// Extract the external ID
		externalID := ""
		if id, ok := supRes["id"].(string); ok {
			externalID = id
		}

		// Persist locally using external id
		ctxLogger.Debugf("Registering user locally with external ID: %s", externalID)

		// First check if user already exists locally to avoid conflicts
		existingUser, _ := h.userService.GetUserByEmail(c, userRequest.Email)
		if existingUser != nil {
			ctxLogger.Info("User already exists locally, returning existing user")
			response.Success(c, http.StatusOK, existingUser, nil, "User already exists")
			return
		}

		// Create new user with external ID
		user, err := h.userService.RegisterUserWithExternalID(c, userRequest, externalID)
		if err != nil {
			ctxLogger.Errorf("Failed to create local user with external ID: %v", err)
			status, code, msg, details := mapDomainError(err)
			response.Error(c, status, code, msg, details)
			return
		}

		ctxLogger.Infof("User created successfully with external ID: %s", externalID)
		response.Success(c, http.StatusCreated, user, nil, "User created successfully")
		return
	}

	// Fallback: create locally
	ctxLogger.Debug("Falling back to local user creation")
	user, err := h.userService.RegisterUser(c, userRequest)
	if err != nil {
		ctxLogger.Errorf("Failed to create user locally: %v", err)
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	ctxLogger.Infof("User created successfully with local auth: %s", user.Email)

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
	ctxLogger := logger.FromGin(c)
	ctxLogger.Info("Login request received")

	var req types.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctxLogger.Errorf("Invalid login request: %v", err)
		response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", err.Error())
		return
	}

	ctxLogger.Debugf("Processing login for email: %s", req.Email)

	supabaseURL := env.GetString("SUPABASE_URL", "")
	supabaseKey := env.GetString("SUPABASE_KEY", "")

	if supabaseURL != "" && supabaseKey != "" {
		ctxLogger.Debug("Using Supabase for authentication")
		// Use the Supabase client for login
		supaClient := supabaseAuth.NewSupabaseClient()

		if !supaClient.IsAvailable() {
			ctxLogger.Error("Supabase client initialization failed")
			response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError, "Supabase client initialization failed", nil)
			return
		}

		// Sign in the user with Supabase
		ctxLogger.Debug("Attempting to sign in with Supabase")
		tokenResp, err := supaClient.SignIn(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			ctxLogger.Warnf("Supabase authentication failed for email %s: %v", req.Email, err)
			response.Error(c, http.StatusUnauthorized, contracts.CodeInvalidCredentials, "Invalid credentials", err.Error())
			return
		}
		ctxLogger.Debug("Successfully authenticated with Supabase")

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
		ctxLogger.Debug("Checking if local user record exists")
		localUser, err := h.userService.GetUserByEmail(c, req.Email)
		if err != nil {
			ctxLogger.Infof("Local user not found for email %s, creating from Supabase data", req.Email)
			// If not found, create local user with external id
			externalID := ""
			if supUser != nil {
				if idv, ok := supUser["id"].(string); ok {
					externalID = idv
					ctxLogger.Debugf("Found external ID: %s", externalID)
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
						ctxLogger.Debugf("Found first name: %s", fn)
					}
					if ln, ok := md["last_name"].(string); ok {
						regReq.LastName = ln
						ctxLogger.Debugf("Found last name: %s", ln)
					}
				}
			}
			ctxLogger.Debug("Registering local user with external ID")
			localUser, err = h.userService.RegisterUserWithExternalID(c, regReq, externalID)
			if err != nil {
				ctxLogger.Errorf("Failed to create local user: %v", err)
			} else {
				ctxLogger.Info("Successfully created local user from external data")
			}
		} else {
			ctxLogger.Debug("Found existing local user")
		}

		// Return local user and tokens with proper type conversion
		ctxLogger.Info("Login successful with Supabase auth")

		// Convert user domain model to response format with proper ID as string
		userResp := UserResponse{
			ID:        localUser.ID.String(),
			FirstName: localUser.FirstName,
			LastName:  localUser.LastName,
			Email:     localUser.Email,
			Age:       int(localUser.Age),
			Name:      localUser.FirstName + " " + localUser.LastName,
		}

		data := map[string]interface{}{
			"user":         userResp,
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
		}
		response.Success(c, http.StatusOK, data, nil, "Login successful")
		return
	}

	// Fallback to local authentication
	ctxLogger.Debug("Falling back to local authentication")
	user, token, err := h.userService.LoginUser(c, req)
	if err != nil {
		ctxLogger.Warnf("Local authentication failed for email %s: %v", req.Email, err)
		status, code, msg, details := mapDomainError(err)
		response.Error(c, status, code, msg, details)
		return
	}

	ctxLogger.Infof("Login successful with local auth for user: %s", user.Email)

	// Convert user domain model to response format with proper ID as string
	userResp := UserResponse{
		ID:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Age:       int(user.Age),
		Name:      user.FirstName + " " + user.LastName,
	}

	data := map[string]interface{}{
		"user":        userResp,
		"accessToken": token, // Using consistent naming (accessToken instead of access_token)
	}
	response.Success(c, http.StatusOK, data, nil, "Login successful")
}
