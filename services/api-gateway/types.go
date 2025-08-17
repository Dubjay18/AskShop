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

// Product represents a product in the system
type Product struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	ImageURL    string  `json:"image_url"`
}
