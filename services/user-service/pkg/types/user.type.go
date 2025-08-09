package types

// UserRegistrationRequest represents a user registration request
type UserRegistrationRequest struct {
	ID        int64  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Password  string `json:"-"` // Password is never returned in JSON
	Age       int64  `json:"age"`
}

// Login request
type UserLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
