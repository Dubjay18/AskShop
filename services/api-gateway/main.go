package main

import (
	"askshop/services/api-gateway/rest"
	"askshop/shared/contracts"
	"askshop/shared/env"
	"askshop/shared/response"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8081")
)

func main() {
	log.Println("Starting API Gateway")

	// Set Gin mode based on environment
	if env.GetString("ENV", "development") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create a default gin router with logger and recovery middleware
	router := gin.Default()

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
			var created User
			if err := userClient.Post(c.Request.Context(), contracts.Routes.Auth.Register, req, &created); err != nil {
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
			if created.Name == "" && created.FirstName != "" { // derive combined name
				created.Name = strings.TrimSpace(created.FirstName + " " + created.LastName)
			}
			response.Success(c, http.StatusCreated, created, nil, "User created successfully")
		})

		// GET /api/v1/users/:identifier (uuid or email)
		users.GET("/:identifier", func(c *gin.Context) {
			ident := c.Param("identifier")
			if ident == "" {
				response.Error(c, http.StatusBadRequest, contracts.CodeUserIdentifierRequired, "User identifier (ID or email) is required", nil)
				return
			}
			var user User
			if err := userClient.Get(c.Request.Context(), "/api/users/"+ident, &user); err != nil {
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
			if user.Name == "" && user.FirstName != "" {
				user.Name = strings.TrimSpace(user.FirstName + " " + user.LastName)
			}
			response.Success(c, http.StatusOK, user, nil, "")
		})
	}

	// Add a root route
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to AskShop API Gateway",
			"version": "1.0.0",
			"endpoints": []string{
				"/api/v1/auth/register - User registration",
				"/api/v1/auth/login - User login",
				"/api/v1/auth/profile - Get user profile (protected)",
				"/api/v1/products - List all products",
				"/api/v1/products/:id - Get product by ID",
				"/api/v1/users - List all users",
				"/api/v1/users/:id - Get user by ID",
				"/health - Health check",
				"/docs - API documentation",
			},
		})
	})

	// Start the server
	log.Printf("API Gateway listening on %s", httpAddr)
	log.Printf("REST API available at http://localhost%s/api/v1", httpAddr)
	if err := router.Run(httpAddr); err != nil {
		log.Printf("HTTP server error %v", err)
	}
}
