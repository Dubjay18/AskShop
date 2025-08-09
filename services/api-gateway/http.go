package main

import (
	"askshop/shared/contracts"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the routes for the API Gateway
func SetupRoutes(router *gin.Engine) {
	// Add middleware
	router.Use(CORSMiddleware())
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware())

	// Basic health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API v1 group
	v1 := router.Group(contracts.Routes.APIBase)
	{
		// Initialize controllers
		productController := NewProductController()
		userController := NewUserController()
		authController := NewAuthController()

		// Public authentication routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
			auth.POST("/refresh", authController.RefreshToken)
			auth.POST("/logout", authController.Logout)
		}

		// Protected authentication routes
		authProtected := v1.Group("/auth")
		authProtected.Use(AuthMiddleware(authController.authClient))
		{
			authProtected.GET(contracts.Routes.Auth.SuffixProfile, authController.GetProfile)
			authProtected.PUT(contracts.Routes.Auth.SuffixProfile, authController.UpdateProfile)
			authProtected.POST(contracts.Routes.Auth.SuffixChangePassword, authController.ChangePassword)
		}

		// Product routes (public for now, can be protected later)
		products := v1.Group("/products")
		{
			products.GET("", productController.GetProducts)
			products.GET(contracts.Routes.Products.SuffixByID, productController.GetProductByID)
			products.POST("", productController.CreateProduct)
			products.PUT(contracts.Routes.Products.SuffixByID, productController.UpdateProduct)
			products.DELETE(contracts.Routes.Products.SuffixByID, productController.DeleteProduct)
		}

		// User routes (public for now, can be protected later)
		users := v1.Group("/users")
		{
			users.GET("", userController.GetUsers)
			users.GET(contracts.Routes.Users.SuffixByID, userController.GetUserByID)
			users.POST("", userController.CreateUser)
		}
	}

	// Documentation endpoint
	router.GET("/docs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "API documentation will be available here",
			"endpoints": map[string]interface{}{
				"authentication": []string{
					"POST " + contracts.Routes.Auth.Register + " - User registration",
					"POST " + contracts.Routes.Auth.Login + " - User login",
					"POST " + contracts.Routes.Auth.Refresh + " - Refresh token",
					"POST " + contracts.Routes.Auth.Logout + " - User logout",
					"GET " + contracts.Routes.Auth.Profile + " - Get user profile (protected)",
					"PUT " + contracts.Routes.Auth.Profile + " - Update user profile (protected)",
					"POST " + contracts.Routes.Auth.ChangePassword + " - Change password (protected)",
				},
				"products": []string{
					"GET " + contracts.Routes.Products.Base + " - List all products",
					"GET " + contracts.Routes.Products.ByID + " - Get product by ID",
					"POST " + contracts.Routes.Products.Base + " - Create product",
					"PUT " + contracts.Routes.Products.ByID + " - Update product",
					"DELETE " + contracts.Routes.Products.ByID + " - Delete product",
				},
				"users": []string{
					"GET " + contracts.Routes.Users.Base + " - List all users",
					"GET " + contracts.Routes.Users.ByID + " - Get user by ID",
					"POST " + contracts.Routes.Users.Base + " - Create user",
				},
			},
		})
	})
}
