package repository

import (
	"askshop/services/cart-service/internal/domain"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ensure PGCartRepository satisfies domain.CartRepository at compile time.
var _ domain.CartRepository = (*PGCartRepository)(nil)

// PGCartRepository implements domain.CartRepository backed by Postgres via GORM.
type PGCartRepository struct {
	db *gorm.DB
}

// NewPGCartRepository creates a new Postgres-backed cart repository.
func NewPGCartRepository(db *gorm.DB) *PGCartRepository {
	return &PGCartRepository{db: db}
}

func (r *PGCartRepository) GetCartByUserID(ctx *gin.Context, userID string) (*domain.Cart, error) {
	var cart domain.Cart
	if err := r.db.WithContext(ctx).Preload("Items").First(&cart, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *PGCartRepository) GetCartBySessionID(ctx *gin.Context, sessionID string) (*domain.Cart, error) {
	var cart domain.Cart
	if err := r.db.WithContext(ctx).Preload("Items").First(&cart, "session_id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *PGCartRepository) CreateCart(ctx *gin.Context, cart *domain.Cart) error {
	return r.db.WithContext(ctx).Create(cart).Error
}

func (r *PGCartRepository) UpdateCart(ctx *gin.Context, cart *domain.Cart) error {
	return r.db.WithContext(ctx).Save(cart).Error
}

func (r *PGCartRepository) DeleteCart(ctx *gin.Context, cartID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Cart{}, "id = ?", cartID).Error
}

func (r *PGCartRepository) AddItemToCart(ctx *gin.Context, cartID uuid.UUID, item *domain.CartItem) error {
	item.CartID = cartID
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *PGCartRepository) UpdateCartItem(ctx *gin.Context, itemID uuid.UUID, quantity int) error {
	return r.db.WithContext(ctx).Model(&domain.CartItem{}).Where("id = ?", itemID).Update("quantity", quantity).Error
}

func (r *PGCartRepository) RemoveCartItem(ctx *gin.Context, itemID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.CartItem{}, "id = ?", itemID).Error
}

func (r *PGCartRepository) GetCartItem(ctx *gin.Context, itemID uuid.UUID) (*domain.CartItem, error) {
	var item domain.CartItem
	if err := r.db.WithContext(ctx).First(&item, "id = ?", itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PGCartRepository) ClearCartItems(ctx *gin.Context, cartID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.CartItem{}, "cart_id = ?", cartID).Error
}

func (r *PGCartRepository) GetCartWithItems(ctx *gin.Context, cartID uuid.UUID) (*domain.Cart, error) {
	var cart domain.Cart
	if err := r.db.WithContext(ctx).Preload("Items").First(&cart, "id = ?", cartID).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *PGCartRepository) SaveItemForLater(ctx *gin.Context, savedItem *domain.SavedItem) error {
	return r.db.WithContext(ctx).Create(savedItem).Error
}

func (r *PGCartRepository) GetSavedItems(ctx *gin.Context, userID string) ([]domain.SavedItem, error) {
	var items []domain.SavedItem
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("priority DESC, created_at DESC").Find(&items).Error
	return items, err
}

func (r *PGCartRepository) RemoveSavedItem(ctx *gin.Context, userID string, productID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.SavedItem{}, "user_id = ? AND product_id = ?", userID, productID).Error
}

func (r *PGCartRepository) MoveSavedItemToCart(ctx *gin.Context, userID string, savedItemID uuid.UUID) error {
	var saved domain.SavedItem
	if err := r.db.WithContext(ctx).First(&saved, "id = ? AND user_id = ?", savedItemID, userID).Error; err != nil {
		return err
	}

	var cart domain.Cart
	if err := r.db.WithContext(ctx).First(&cart, "user_id = ?", userID).Error; err != nil {
		return errors.New("cart not found for user")
	}

	item := &domain.CartItem{
		CartID:      cart.ID,
		ProductID:   saved.ProductID,
		ProductSKU:  saved.ProductSKU,
		ProductName: saved.ProductName,
		UnitPrice:   saved.UnitPrice,
		Currency:    saved.Currency,
		Quantity:    1,
		Variations:  saved.Variations,
		Notes:       saved.Notes,
	}
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).Delete(&domain.SavedItem{}, "id = ?", savedItemID).Error
}

func (r *PGCartRepository) GetAbandonedCarts(ctx *gin.Context, since *time.Time) ([]domain.Cart, error) {
	var carts []domain.Cart
	q := r.db.WithContext(ctx).Where("status = ?", "active")
	if since != nil {
		q = q.Where("updated_at < ?", *since)
	}
	err := q.Preload("Items").Find(&carts).Error
	return carts, err
}

func (r *PGCartRepository) GetExpiredCarts(ctx *gin.Context) ([]domain.Cart, error) {
	var carts []domain.Cart
	err := r.db.WithContext(ctx).Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Find(&carts).Error
	return carts, err
}

func (r *PGCartRepository) CleanupExpiredCarts(ctx *gin.Context) (int64, error) {
	result := r.db.WithContext(ctx).Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Delete(&domain.Cart{})
	return result.RowsAffected, result.Error
}
