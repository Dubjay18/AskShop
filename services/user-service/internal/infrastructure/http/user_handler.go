package http

import (
	"askshop/services/user-service/internal/service"
	"askshop/services/user-service/pkg/types"
	"askshop/shared/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
	api := router.Group("/api")
	{
		users := api.Group("/users")
		{
			users.GET("/:identifier", h.GetUser)
			users.POST("/", h.CreateUser)
		}
	}
}

// GetUser handles both ID and email-based user lookup
// Example usage:
// GET /api/users/123e4567-e89b-12d3-a456-426614174000 (UUID)
// GET /api/users/user@example.com (email)
func (h *UserHandler) GetUser(c *gin.Context) {
	identifier := c.Param("identifier")

	if identifier == "" {
		response.Error(c, http.StatusBadRequest, "USER_IDENTIFIER_REQUIRED", "User identifier (ID or email) is required", nil)
		return
	}

	user, err := h.userService.GetUserByIDOrEmail(c, identifier)
	if err != nil {
		response.Error(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil)
		return
	}

	response.Success(c, http.StatusOK, user, nil, "")
}

// GetUserById handles ID-specific user lookup
func (h *UserHandler) GetUserById(c *gin.Context) {
	userID := c.Param("id")

	if userID == "" {
		response.Error(c, http.StatusBadRequest, "USER_ID_REQUIRED", "User ID is required", nil)
		return
	}

	user, err := h.userService.GetUserById(c, userID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil)
		return
	}

	response.Success(c, http.StatusOK, user, nil, "")
}

// GetUserByEmail handles email-specific user lookup
func (h *UserHandler) GetUserByEmail(c *gin.Context) {
	email := c.Query("email")

	if email == "" {
		response.Error(c, http.StatusBadRequest, "EMAIL_REQUIRED", "Email is required", nil)
		return
	}

	user, err := h.userService.GetUserByEmail(c, email)
	if err != nil {
		response.Error(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil)
		return
	}

	response.Success(c, http.StatusOK, user, nil, "")
}

// CreateUser handles user creation
func (h *UserHandler) CreateUser(c *gin.Context) {
	var userRequest types.UserRegistrationRequest
	if err := c.ShouldBindJSON(&userRequest); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body", err.Error())
		return
	}

	user, err := h.userService.RegisterUser(c, userRequest)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "USER_CREATION_FAILED", "Failed to create user", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, user, nil, "User created successfully")
}
