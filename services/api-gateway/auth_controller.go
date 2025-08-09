package main

import (
	"askshop/services/api-gateway/rest"
	"askshop/shared/response"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthController handles authentication-related API requests
type AuthController struct {
	authClient *rest.ServiceClient
}

// NewAuthController creates a new auth controller
func NewAuthController() *AuthController {
	return &AuthController{
		authClient: rest.NewServiceClient("auth"),
	}
}

// Register handles user registration
func (c *AuthController) Register(ctx *gin.Context) {
	var registerRequest struct {
		Email     string `json:"email" binding:"required,email"`
		Username  string `json:"username" binding:"required,min=3"`
		Password  string `json:"password" binding:"required,min=6"`
		FirstName string `json:"first_name" binding:"required"`
		LastName  string `json:"last_name" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&registerRequest); err != nil {
		response.Error(ctx, http.StatusBadRequest, "AUTH_INVALID_BODY", "Invalid request body", err.Error())
		return
	}
	var authResponse interface{}
	if err := c.authClient.Post(ctx.Request.Context(), "/api/auth/register", registerRequest, &authResponse); err != nil {
		code := http.StatusInternalServerError
		errCode := "AUTH_REGISTER_FAILED"
		if strings.Contains(err.Error(), "already exists") {
			code = http.StatusConflict
			errCode = "AUTH_USER_EXISTS"
		}
		response.Error(ctx, code, errCode, fmt.Sprintf("Registration failed: %v", err), nil)
		return
	}
	response.Success(ctx, http.StatusCreated, authResponse, nil, "Registration successful")
}

// Login handles user login
func (c *AuthController) Login(ctx *gin.Context) {
	var loginRequest struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&loginRequest); err != nil {
		response.Error(ctx, http.StatusBadRequest, "AUTH_INVALID_BODY", "Invalid request body", err.Error())
		return
	}
	var authResponse interface{}
	if err := c.authClient.Post(ctx.Request.Context(), "/api/auth/login", loginRequest, &authResponse); err != nil {
		response.Error(ctx, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", "Invalid credentials", nil)
		return
	}
	response.Success(ctx, http.StatusOK, authResponse, nil, "Login successful")
}

// RefreshToken handles token refresh
func (c *AuthController) RefreshToken(ctx *gin.Context) {
	var refreshRequest struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&refreshRequest); err != nil {
		response.Error(ctx, http.StatusBadRequest, "AUTH_INVALID_BODY", "Invalid request body", err.Error())
		return
	}
	var authResponse interface{}
	if err := c.authClient.Post(ctx.Request.Context(), "/api/auth/refresh", refreshRequest, &authResponse); err != nil {
		response.Error(ctx, http.StatusUnauthorized, "AUTH_REFRESH_FAILED", "Invalid refresh token", nil)
		return
	}
	response.Success(ctx, http.StatusOK, authResponse, nil, "Token refreshed")
}

// Logout handles user logout
func (c *AuthController) Logout(ctx *gin.Context) {
	if err := c.authClient.Post(ctx.Request.Context(), "/api/auth/logout", nil, nil); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "AUTH_LOGOUT_FAILED", "Logout failed", nil)
		return
	}
	response.Success(ctx, http.StatusOK, gin.H{"logout": true}, nil, "Logout successful")
}

// GetProfile handles getting user profile
func (c *AuthController) GetProfile(ctx *gin.Context) {
	var profile interface{}
	if err := c.authClient.Get(ctx.Request.Context(), "/api/profile", &profile); err != nil {
		response.Error(ctx, http.StatusNotFound, "AUTH_PROFILE_NOT_FOUND", "Profile not found", nil)
		return
	}
	response.Success(ctx, http.StatusOK, profile, nil, "")
}

// UpdateProfile handles updating user profile
func (c *AuthController) UpdateProfile(ctx *gin.Context) {
	var profileRequest interface{}
	if err := ctx.ShouldBindJSON(&profileRequest); err != nil {
		response.Error(ctx, http.StatusBadRequest, "AUTH_INVALID_BODY", "Invalid request body", err.Error())
		return
	}
	var updatedProfile interface{}
	if err := c.authClient.Put(ctx.Request.Context(), "/api/profile", profileRequest, &updatedProfile); err != nil {
		code := http.StatusInternalServerError
		errCode := "AUTH_PROFILE_UPDATE_FAILED"
		if strings.Contains(err.Error(), "already exists") {
			code = http.StatusConflict
			errCode = "AUTH_PROFILE_CONFLICT"
		}
		response.Error(ctx, code, errCode, "Profile update failed", nil)
		return
	}
	response.Success(ctx, http.StatusOK, updatedProfile, nil, "Profile updated")
}

// ChangePassword handles password change
func (c *AuthController) ChangePassword(ctx *gin.Context) {
	var changePasswordRequest struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := ctx.ShouldBindJSON(&changePasswordRequest); err != nil {
		response.Error(ctx, http.StatusBadRequest, "AUTH_INVALID_BODY", "Invalid request body", err.Error())
		return
	}
	if err := c.authClient.Post(ctx.Request.Context(), "/api/change-password", changePasswordRequest, nil); err != nil {
		code := http.StatusInternalServerError
		errCode := "AUTH_PASSWORD_CHANGE_FAILED"
		if strings.Contains(err.Error(), "invalid old password") {
			code = http.StatusBadRequest
			errCode = "AUTH_INVALID_OLD_PASSWORD"
		}
		response.Error(ctx, code, errCode, "Password change failed", nil)
		return
	}
	response.Success(ctx, http.StatusOK, gin.H{"password_changed": true}, nil, "Password changed")
}

// AuthMiddleware validates JWT tokens using the auth service
func AuthMiddleware(authClient *rest.ServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "AUTH_HEADER_REQUIRED", "Authorization header required", nil)
			c.Abort()
			return
		}
		var tokenClaims interface{}
		if err := authClient.Post(c.Request.Context(), "/api/auth/validate", nil, &tokenClaims); err != nil {
			response.Error(c, http.StatusUnauthorized, "AUTH_INVALID_TOKEN", "Invalid or expired token", nil)
			c.Abort()
			return
		}
		c.Set("token_claims", tokenClaims)
		c.Next()
	}
}
