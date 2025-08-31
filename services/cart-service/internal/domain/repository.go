package domain

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"time"
)

// CartRepository defines the data access interface for cart operations
type CartRepository interface {
	// Cart operations
	GetCartByUserID(ctx *gin.Context, userID string) (*Cart, error)
	GetCartBySessionID(ctx *gin.Context, sessionID string) (*Cart, error)
	CreateCart(ctx *gin.Context, cart *Cart) error
	UpdateCart(ctx *gin.Context, cart *Cart) error
	DeleteCart(ctx *gin.Context, cartID uuid.UUID) error

	// Cart item operations
	AddItemToCart(ctx *gin.Context, cartID uuid.UUID, item *CartItem) error
	UpdateCartItem(ctx *gin.Context, itemID uuid.UUID, quantity int) error
	RemoveCartItem(ctx *gin.Context, itemID uuid.UUID) error
	GetCartItem(ctx *gin.Context, itemID uuid.UUID) (*CartItem, error)

	// Bulk operations
	ClearCartItems(ctx *gin.Context, cartID uuid.UUID) error
	GetCartWithItems(ctx *gin.Context, cartID uuid.UUID) (*Cart, error)

	// Saved items operations
	SaveItemForLater(ctx *gin.Context, savedItem *SavedItem) error
	GetSavedItems(ctx *gin.Context, userID string) ([]SavedItem, error)
	RemoveSavedItem(ctx *gin.Context, userID string, productID uuid.UUID) error
	MoveSavedItemToCart(ctx *gin.Context, userID string, savedItemID uuid.UUID) error

	// Analytics and cleanup
	GetAbandonedCarts(ctx *gin.Context, since *time.Time) ([]Cart, error)
	GetExpiredCarts(ctx *gin.Context) ([]Cart, error)
	CleanupExpiredCarts(ctx *gin.Context) (int64, error)
}
