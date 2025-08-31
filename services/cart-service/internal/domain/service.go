package domain

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"time"
)

// CartService defines the business logic interface for cart operations
type CartService interface {
	// Cart management
	GetOrCreateCart(ctx *gin.Context, userID string, sessionID string) (*Cart, error)
	GetCart(ctx *gin.Context, userID string) (*Cart, error)
	GetCartSummary(ctx *gin.Context, userID string) (*CartSummary, error)
	ClearCart(ctx *gin.Context, userID string) error
	ConvertCart(ctx *gin.Context, userID string) error // Mark cart as converted to order

	// Item management
	AddItem(ctx *gin.Context, userID string, productID uuid.UUID, quantity int, variations datatypes.JSONMap) (*Cart, error)
	UpdateItemQuantity(ctx *gin.Context, userID string, itemID uuid.UUID, quantity int) (*Cart, error)
	RemoveItem(ctx *gin.Context, userID string, itemID uuid.UUID) error

	// Wishlist/Save for later
	SaveForLater(ctx *gin.Context, userID string, itemID uuid.UUID) error
	GetSavedItems(ctx *gin.Context, userID string) ([]SavedItem, error)
	MoveToCart(ctx *gin.Context, userID string, savedItemID uuid.UUID) error
	RemoveSavedItem(ctx *gin.Context, userID string, savedItemID uuid.UUID) error

	// Cart migration (anonymous to authenticated)
	MergeAnonymousCart(ctx *gin.Context, sessionID string, userID string) (*Cart, error)

	// Analytics and maintenance
	GetAbandonedCarts(ctx *gin.Context, since time.Time) ([]Cart, error)
	CleanupExpiredCarts(ctx *gin.Context) (int64, error)

	// Validation
	ValidateCartForCheckout(ctx *gin.Context, userID string) (*CartValidationResult, error)
}

// CartValidationResult represents the result of cart validation for checkout
type CartValidationResult struct {
	IsValid          bool                  `json:"isValid"`
	Errors           []string              `json:"errors"`
	Warnings         []string              `json:"warnings"`
	UnavailableItems []uuid.UUID           `json:"unavailableItems"`
	PriceChanges     []CartItemPriceChange `json:"priceChanges"`
	UpdatedCart      *Cart                 `json:"updatedCart,omitempty"`
}

// CartItemPriceChange represents a price change for a cart item
type CartItemPriceChange struct {
	ItemID      uuid.UUID `json:"itemId"`
	ProductID   uuid.UUID `json:"productId"`
	ProductName string    `json:"productName"`
	OldPrice    float64   `json:"oldPrice"`
	NewPrice    float64   `json:"newPrice"`
	Difference  float64   `json:"difference"`
}

// AddItemRequest represents a request to add an item to cart
type AddItemRequest struct {
	ProductID  uuid.UUID         `json:"productId" binding:"required"`
	Quantity   int               `json:"quantity" binding:"required,min=1"`
	Variations datatypes.JSONMap `json:"variations"`
	Notes      string            `json:"notes"`
}

// UpdateItemRequest represents a request to update a cart item
type UpdateItemRequest struct {
	Quantity int    `json:"quantity" binding:"required,min=0"` // 0 means remove
	Notes    string `json:"notes"`
}
