package main

import (
	"askshop/services/api-gateway/rest"
	"askshop/shared/response"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ProductController handles product-related API requests
type ProductController struct {
	productClient *rest.ServiceClient
}

// NewProductController creates a new product controller
func NewProductController() *ProductController {
	return &ProductController{
		productClient: rest.NewServiceClient("product"),
	}
}

// GetProducts retrieves a list of products
func (c *ProductController) GetProducts(ctx *gin.Context) {
	// Parse query parameters for filtering, pagination, etc.
	limit := 10
	page := 1
	var err error

	if limitParam := ctx.Query("limit"); limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil {
			response.Error(ctx, http.StatusBadRequest, "PRODUCT_INVALID_LIMIT", "Invalid limit parameter", nil)
			return
		}
	}

	if pageParam := ctx.Query("page"); pageParam != "" {
		page, err = strconv.Atoi(pageParam)
		if err != nil {
			response.Error(ctx, http.StatusBadRequest, "PRODUCT_INVALID_PAGE", "Invalid page parameter", nil)
			return
		}
	}

	// Build query path
	path := fmt.Sprintf("/api/products?limit=%d&page=%d", limit, page)

	// Add any additional filters
	if category := ctx.Query("category"); category != "" {
		path += fmt.Sprintf("&category=%s", category)
	}

	if search := ctx.Query("search"); search != "" {
		path += fmt.Sprintf("&search=%s", search)
	}

	// Get products from product service
	var products []Product
	if err = c.productClient.Get(ctx.Request.Context(), path, &products); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "PRODUCT_FETCH_FAILED", fmt.Sprintf("Error fetching products: %v", err), nil)
		return
	}

	// Return products
	meta := gin.H{
		"page":  page,
		"limit": limit,
		"total": len(products), // This should be replaced with actual total count from the service
	}
	response.Success(ctx, http.StatusOK, products, meta, "")
}

// GetProductByID retrieves a product by ID
func (c *ProductController) GetProductByID(ctx *gin.Context) {
	id := ctx.Param("id")

	// Get product from product service
	var product Product
	if err := c.productClient.Get(ctx.Request.Context(), fmt.Sprintf("/api/products/%s", id), &product); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "PRODUCT_FETCH_FAILED", fmt.Sprintf("Error fetching product: %v", err), nil)
		return
	}

	// Return product
	response.Success(ctx, http.StatusOK, product, nil, "")
}

// CreateProduct creates a new product
func (c *ProductController) CreateProduct(ctx *gin.Context) {
	var product Product
	if err := ctx.ShouldBindJSON(&product); err != nil {
		response.Error(ctx, http.StatusBadRequest, "PRODUCT_INVALID_BODY", "Invalid request body", err.Error())
		return
	}

	// Create product in product service
	var createdProduct Product
	if err := c.productClient.Post(ctx.Request.Context(), "/api/products", product, &createdProduct); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "PRODUCT_CREATE_FAILED", fmt.Sprintf("Error creating product: %v", err), nil)
		return
	}

	// Return created product
	response.Success(ctx, http.StatusCreated, createdProduct, nil, "Product created")
}

// UpdateProduct updates an existing product
func (c *ProductController) UpdateProduct(ctx *gin.Context) {
	id := ctx.Param("id")

	var product Product
	if err := ctx.ShouldBindJSON(&product); err != nil {
		response.Error(ctx, http.StatusBadRequest, "PRODUCT_INVALID_BODY", "Invalid request body", err.Error())
		return
	}

	// Update product in product service
	var updatedProduct Product
	if err := c.productClient.Put(ctx.Request.Context(), fmt.Sprintf("/api/products/%s", id), product, &updatedProduct); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "PRODUCT_UPDATE_FAILED", fmt.Sprintf("Error updating product: %v", err), nil)
		return
	}

	// Return updated product
	response.Success(ctx, http.StatusOK, updatedProduct, nil, "Product updated")
}

// DeleteProduct deletes a product
func (c *ProductController) DeleteProduct(ctx *gin.Context) {
	id := ctx.Param("id")

	// Delete product from product service
	if err := c.productClient.Delete(ctx.Request.Context(), fmt.Sprintf("/api/products/%s", id), nil); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "PRODUCT_DELETE_FAILED", fmt.Sprintf("Error deleting product: %v", err), nil)
		return
	}

	// Return success
	response.Success(ctx, http.StatusOK, gin.H{"deleted": true}, nil, "Product deleted")
}

// UserController handles user-related API requests
type UserController struct {
	userClient *rest.ServiceClient
}

// NewUserController creates a new user controller
func NewUserController() *UserController {
	return &UserController{
		userClient: rest.NewServiceClient("user"),
	}
}

// GetUsers retrieves a list of users
func (c *UserController) GetUsers(ctx *gin.Context) {
	// Parse query parameters for filtering, pagination, etc.
	limit := 10
	page := 1
	var err error

	if limitParam := ctx.Query("limit"); limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil {
			response.Error(ctx, http.StatusBadRequest, "USER_INVALID_LIMIT", "Invalid limit parameter", nil)
			return
		}
	}

	if pageParam := ctx.Query("page"); pageParam != "" {
		page, err = strconv.Atoi(pageParam)
		if err != nil {
			response.Error(ctx, http.StatusBadRequest, "USER_INVALID_PAGE", "Invalid page parameter", nil)
			return
		}
	}

	// Build query path
	path := fmt.Sprintf("/api/users?limit=%d&page=%d", limit, page)

	// Get users from user service
	var users []User
	if err = c.userClient.Get(ctx.Request.Context(), path, &users); err != nil {
		// For now, just use mock data since we don't have the user service yet
		users = []User{
			{ID: 1, Name: "User 1", Email: "user1@example.com"},
			{ID: 2, Name: "User 2", Email: "user2@example.com"},
		}
	}

	// Return users
	meta := gin.H{
		"page":  page,
		"limit": limit,
		"total": len(users), // This should be replaced with actual total count from the service
	}
	response.Success(ctx, http.StatusOK, users, meta, "")
}

// GetUserByID retrieves a user by ID
func (c *UserController) GetUserByID(ctx *gin.Context) {
	id := ctx.Param("id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, "USER_INVALID_ID", "Invalid user ID", nil)
		return
	}

	// Get user from user service
	var user User
	if err = c.userClient.Get(ctx.Request.Context(), fmt.Sprintf("/api/users/%s", id), &user); err != nil {
		// For now, just use mock data since we don't have the user service yet
		user = User{
			ID:    idInt,
			Name:  fmt.Sprintf("User %d", idInt),
			Email: fmt.Sprintf("user%d@example.com", idInt),
		}
	}

	// Return user
	response.Success(ctx, http.StatusOK, user, nil, "")
}

// CreateUser creates a new user
func (c *UserController) CreateUser(ctx *gin.Context) {
	var user User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		response.Error(ctx, http.StatusBadRequest, "USER_INVALID_BODY", "Invalid request body", err.Error())
		return
	}

	// Create user in user service
	var createdUser User
	if err := c.userClient.Post(ctx.Request.Context(), "/api/users", user, &createdUser); err != nil {
		// For now, just use mock data since we don't have the user service yet
		createdUser = User{
			ID:    3,
			Name:  user.Name,
			Email: user.Email,
		}
	}

	// Return created user
	response.Success(ctx, http.StatusCreated, createdUser, nil, "User created")
}
