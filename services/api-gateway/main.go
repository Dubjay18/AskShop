package main

import (
	"askshop/services/api-gateway/rest"
	"askshop/shared/contracts"
	"askshop/shared/env"
	"askshop/shared/response"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8081")
	logger   *Logger
)

func init() {
	// Initialize logger with environment-specific configuration
	logConfig := NewLogConfig()
	logger = NewLogger(logConfig)
}

func main() {

	logger.Info("Starting API Gateway")

	// Set Gin mode based on environment
	isProd := env.GetString("ENV", "development") == "production"
	if isProd {
		gin.SetMode(gin.ReleaseMode)
		logger.Info("Running in production mode")
	} else {
		logger.Info("Running in development mode")
	}

	// Create a new gin router without default middleware
	router := gin.New()

	// Add recovery middleware to prevent server crashes
	router.Use(gin.Recovery())

	// Global middleware
	router.Use(RequestIDMiddleware(), LoggingMiddleware(), CORSMiddleware())

	// Service clients
	userClient := rest.NewServiceClient("user")

	// API v1 group
	v1 := router.Group(contracts.Routes.APIBase)

	// Users routes
	users := v1.Group("/users")
	auth := v1.Group("/auth")
	{
		// Service diagnostics - forward to user service diagnostics endpoint
		if env.GetString("ENV", "development") != "production" {
			auth.GET("/service-diagnostics", func(c *gin.Context) {
				logger.Info("Forwarding diagnostics request to user-service")

				// Call user service diagnostics endpoint
				var diagnosticsResponse map[string]interface{}
				if err := userClient.Get(c.Request.Context(), "/api/v1/auth/diagnostics", &diagnosticsResponse); err != nil {
					logger.WithField("error", err.Error()).Error("Error fetching diagnostics from user service")
					response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError,
						"Error fetching diagnostics from user service", err.Error())
					return
				}

				logger.Info("Successfully retrieved diagnostics from user service")
				response.Success(c, http.StatusOK, diagnosticsResponse, nil, "User service diagnostics")
			})
		}

		auth.POST("/register", func(c *gin.Context) {
			var req UserRegistrationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				// Attempt to collect validation errors
				var details interface{} = err.Error()
				if verrs, ok := err.(validator.ValidationErrors); ok {
					fe := make([]map[string]string, 0, len(verrs))
					for _, v := range verrs {
						fe = append(fe, map[string]string{
							"field": v.Field(),
							"tag":   v.Tag(),
							"value": v.Param(),
						})
					}
					details = fe
				}
				response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", details)
				return
			}

			// First get the raw response as a map to handle potential type mismatches
			var rawResponse map[string]interface{}
			if err := userClient.Post(c.Request.Context(), contracts.Routes.Auth.Register, req, &rawResponse); err != nil {
				lower := strings.ToLower(err.Error())
				status := http.StatusConflict
				code := contracts.CodeUserAlreadyExists
				msg := "User already exists"
				if !strings.Contains(lower, "exists") {
					status = http.StatusInternalServerError
					code = contracts.CodeInternalServerError
					msg = "User creation failed"
				}
				response.Error(c, status, code, msg, err.Error())
				return
			}

			// Convert the raw response to our User type using the utility function
			created, err := ConvertUserResponse(rawResponse)
			if err != nil {
				logger.WithField("error", err.Error()).Error("Error converting user response")
				response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError,
					"Error processing user data", err.Error())
				return
			}

			// Ensure name is set
			if created.Name == "" && created.FirstName != "" {
				created.Name = strings.TrimSpace(created.FirstName + " " + created.LastName)
			}

			response.Success(c, http.StatusCreated, created, nil, "User created successfully")
		})

		auth.POST("/login", func(c *gin.Context) {
			var req UserLoginRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				// Attempt to collect validation errors
				var details interface{} = err.Error()
				if verrs, ok := err.(validator.ValidationErrors); ok {
					fe := make([]map[string]string, 0, len(verrs))
					for _, v := range verrs {
						fe = append(fe, map[string]string{
							"field": v.Field(),
							"tag":   v.Tag(),
							"value": v.Param(),
						})
					}
					details = fe
				}
				response.Error(c, http.StatusBadRequest, contracts.CodeInvalidRequestBody, "Invalid request body", details)
				return
			}

			// First get the raw response as a map to handle potential type mismatches
			var rawResponse map[string]interface{}
			if err := userClient.Post(c.Request.Context(), contracts.Routes.Auth.Login, req, &rawResponse); err != nil {
				lower := strings.ToLower(err.Error())
				status := http.StatusUnauthorized
				code := contracts.CodeInvalidCredentials
				msg := "Invalid email or password"
				if strings.Contains(lower, "internal") {
					status = http.StatusInternalServerError
					code = contracts.CodeInternalServerError
					msg = "Internal error during login"
				}
				response.Error(c, status, code, msg, err.Error())
				return
			}

			// Convert the raw response to our AuthResponse type using the utility function
			resp, err := ConvertAuthResponse(rawResponse)
			if err != nil {
				logger.WithField("error", err.Error()).Error("Error converting auth response")
				response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError,
					"Error processing authentication data", err.Error())
				return
			}

			response.Success(c, http.StatusOK, resp, nil, "Login successful")
		})

		// GET /api/v1/users/:identifier (uuid or email)
		users.GET("/:identifier", func(c *gin.Context) {
			ident := c.Param("identifier")
			if ident == "" {
				response.Error(c, http.StatusBadRequest, contracts.CodeUserIdentifierRequired, "User identifier (ID or email) is required", nil)
				return
			}

			// First get the raw response as a map to handle potential type mismatches
			var rawResponse map[string]interface{}
			if err := userClient.Get(c.Request.Context(), "/api/users/"+ident, &rawResponse); err != nil {
				lower := strings.ToLower(err.Error())
				code := contracts.CodeUserNotFound
				status := http.StatusNotFound
				msg := "User not found"
				if strings.Contains(lower, "internal") {
					code = contracts.CodeInternalServerError
					status = http.StatusInternalServerError
					msg = "Internal error retrieving user"
				}
				response.Error(c, status, code, msg, err.Error())
				return
			}

			// Convert the raw response to our User type using the utility function
			user, err := ConvertUserResponse(rawResponse)
			if err != nil {
				logger.WithField("error", err.Error()).Error("Error converting user response")
				response.Error(c, http.StatusInternalServerError, contracts.CodeInternalServerError,
					"Error processing user data", err.Error())
				return
			}

			if user.Name == "" && user.FirstName != "" {
				user.Name = strings.TrimSpace(user.FirstName + " " + user.LastName)
			}
			response.Success(c, http.StatusOK, user, nil, "")
		})
	}

	// Add a root route
	router.GET("/", func(c *gin.Context) {
		// Get environment to show/hide development-only endpoints
		isDev := env.GetString("ENV", "development") != "production"

		endpoints := []string{
			"/api/v1/auth/register - User registration",
			"/api/v1/auth/login - User login",
			"/api/v1/auth/profile - Get user profile (protected)",
			"/api/v1/products - List all products",
			"/api/v1/products/:id - Get product by ID",
			"/api/v1/users - List all users",
			"/api/v1/users/:id - Get user by ID",
			"/health - Health check",
			"/docs - API documentation",
		}

		// Add development-only endpoints
		if isDev {
			devEndpoints := []string{
				"/api/v1/auth/service-diagnostics - Authentication diagnostics from user service (dev only)",
			}
			endpoints = append(endpoints, devEndpoints...)
		}

		c.JSON(200, gin.H{
			"message":     "Welcome to AskShop API Gateway",
			"version":     "1.0.0",
			"environment": env.GetString("ENV", "development"),
			"endpoints":   endpoints,
		})
	})

	// Start the server
	logger.WithField("port", httpAddr).Info("API Gateway listening")
	logger.WithField("url", fmt.Sprintf("http://localhost%s/api/v1", httpAddr)).Info("REST API available")

	if err := router.Run(httpAddr); err != nil {
		logger.WithField("error", err.Error()).Fatal("HTTP server error")
	}
}
