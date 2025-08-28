package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CartRepository defines the data access interface for cart operations
type CartRepository interface {
	// Cart operations
	GetCartByUserID(ctx context.Context, userID string) (*Cart, error)
	GetCartBySessionID(ctx context.Context, sessionID string) (*Cart, error)
	CreateCart(ctx context.Context, cart *Cart) error
	UpdateCart(ctx context.Context, cart *Cart) error
	DeleteCart(ctx context.Context, cartID uuid.UUID) error

	// Cart item operations
	AddItemToCart(ctx context.Context, cartID uuid.UUID, item *CartItem) error
	UpdateCartItem(ctx context.Context, itemID uuid.UUID, quantity int) error
	RemoveCartItem(ctx context.Context, itemID uuid.UUID) error
	GetCartItem(ctx context.Context, itemID uuid.UUID) (*CartItem, error)

	// Bulk operations
	ClearCartItems(ctx context.Context, cartID uuid.UUID) error
	GetCartWithItems(ctx context.Context, cartID uuid.UUID) (*Cart, error)

	// Saved items operations
	SaveItemForLater(ctx context.Context, savedItem *SavedItem) error
	GetSavedItems(ctx context.Context, userID string) ([]SavedItem, error)
	RemoveSavedItem(ctx context.Context, userID string, productID uuid.UUID) error
	MoveSavedItemToCart(ctx context.Context, userID string, savedItemID uuid.UUID) error

	// Analytics and cleanup
	GetAbandonedCarts(ctx context.Context, since *time.Time) ([]Cart, error)
	GetExpiredCarts(ctx context.Context) ([]Cart, error)
	CleanupExpiredCarts(ctx context.Context) (int64, error)
}
