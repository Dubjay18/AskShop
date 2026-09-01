package main

// APIResponse represents a generic API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// User represents a user returned from the user service (UUID IDs, separated names)
type User struct {
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

// UserRegistrationRequest forwarded to user service
type UserRegistrationRequest struct {
	FirstName string `json:"firstName" binding:"required,min=2"`
	LastName  string `json:"lastName" binding:"required,min=2"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	Age       int    `json:"age,omitempty" binding:"omitempty,gte=0,lte=130"`
}

// UserLoginRequest forwarded to user service
type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents the response from user service on successful login/registration
type AuthResponse struct {
	User         User   `json:"user"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

// Product represents a product returned from product-service
type Product struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Slug          string   `json:"slug"`
	SKU           string   `json:"sku"`
	Description   string   `json:"description"`
	Currency      string   `json:"currency"`
	Status        string   `json:"status"`
	Tags          []string `json:"tags"`
	PriceCents    int64    `json:"priceCents"`
	StockQuantity int      `json:"stockQuantity"`
}
