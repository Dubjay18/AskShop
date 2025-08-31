package service

import (
	"askshop/services/cart-service/internal/domain"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// CartServiceImpl implements the CartService interface
type CartServiceImpl struct {
	repo    domain.CartRepository
	manager *domain.CartManager
	config  domain.CartConfig
}

// NewCartService creates a new cart service
func NewCartService(repo domain.CartRepository, config domain.CartConfig) domain.CartService {
	return &CartServiceImpl{
		repo:    repo,
		manager: domain.NewCartManager(config),
		config:  config,
	}
}

// GetOrCreateCart gets an existing cart or creates a new one
func (s *CartServiceImpl) GetOrCreateCart(ctx *gin.Context, userID string, sessionID string) (*domain.Cart, error) {
	// Try to get existing cart by user ID
	cart, err := s.repo.GetCartByUserID(ctx, userID)
	if err == nil {
		return cart, nil
	}

	// If user doesn't have a cart, try to get by session ID (for anonymous users)
	if sessionID != "" {
		cart, err = s.repo.GetCartBySessionID(ctx, sessionID)
		if err == nil {
			// Update cart with user ID if user just logged in
			if userID != "" && cart.UserID == "" {
				cart.UserID = userID
				cart.SessionID = "" // Clear session ID as it's no longer needed
				if err := s.repo.UpdateCart(ctx, cart); err != nil {
					return nil, fmt.Errorf("failed to update cart with user ID: %w", err)
				}
			}
			return cart, nil
		}
	}

	// Create new cart
	cart = &domain.Cart{
		UserID:    userID,
		SessionID: sessionID,
		Status:    "active",
		Currency:  s.config.Currency,
		Items:     []domain.CartItem{},
	}

	// Set expiry
	expiry := time.Now().Add(s.config.DefaultCartExpiry)
	cart.ExpiresAt = &expiry

	if err := s.repo.CreateCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to create cart: %w", err)
	}

	return cart, nil
}

// GetCart gets a cart by user ID
func (s *CartServiceImpl) GetCart(ctx *gin.Context, userID string) (*domain.Cart, error) {
	cart, err := s.repo.GetCartByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// Check if cart is expired
	if cart.IsExpired() {
		return nil, domain.ErrCartExpired
	}

	return cart, nil
}

// GetCartSummary gets cart summary with totals
func (s *CartServiceImpl) GetCartSummary(ctx *gin.Context, userID string) (*domain.CartSummary, error) {
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	summary := s.manager.CalculateCartSummary(cart)
	return &summary, nil
}

// AddItem adds an item to the cart
func (s *CartServiceImpl) AddItem(ctx *gin.Context, userID string, productID uuid.UUID, quantity int, variations datatypes.JSONMap) (*domain.Cart, error) {
	// Get or create cart
	cart, err := s.GetOrCreateCart(ctx, userID, "")
	if err != nil {
		return nil, err
	}

	// TODO: Get product details from product service
	// For now, we'll use placeholder data
	productName := fmt.Sprintf("Product %s", productID.String()[:8])
	productSKU := fmt.Sprintf("SKU-%s", productID.String()[:8])
	unitPrice := 29.99 // This should come from product service

	// Build cart item
	builder := domain.NewCartItemBuilder(s.config)
	item, err := builder.
		WithProduct(productID, productSKU, productName, unitPrice).
		WithQuantity(quantity).
		WithVariations(variations).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build cart item: %w", err)
	}

	// Check if similar item exists (same product + variations)
	existing := cart.GetItemByProductID(productID, variations)
	if existing != nil {
		// Update quantity of existing item
		newQuantity := existing.Quantity + quantity
		if err := s.manager.ValidateQuantityUpdate(newQuantity); err != nil {
			return nil, err
		}
		return s.UpdateItemQuantity(ctx, userID, existing.ID, newQuantity)
	}

	// Validate adding new item
	if err := s.manager.ValidateAddItem(cart, item); err != nil {
		return nil, err
	}

	// Add item to cart
	item.CartID = cart.ID
	if err := s.repo.AddItemToCart(ctx, cart.ID, item); err != nil {
		return nil, fmt.Errorf("failed to add item to cart: %w", err)
	}

	// Get updated cart
	return s.repo.GetCartWithItems(ctx, cart.ID)
}

// UpdateItemQuantity updates the quantity of a cart item
func (s *CartServiceImpl) UpdateItemQuantity(ctx *gin.Context, userID string, itemID uuid.UUID, quantity int) (*domain.Cart, error) {
	// Validate quantity
	if err := s.manager.ValidateQuantityUpdate(quantity); err != nil {
		return nil, err
	}

	// If quantity is 0, remove the item
	if quantity == 0 {
		return nil, s.RemoveItem(ctx, userID, itemID)
	}

	// Update item
	if err := s.repo.UpdateCartItem(ctx, itemID, quantity); err != nil {
		return nil, fmt.Errorf("failed to update cart item: %w", err)
	}

	// Get cart for the item
	item, err := s.repo.GetCartItem(ctx, itemID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetCartWithItems(ctx, item.CartID)
}

// RemoveItem removes an item from the cart
func (s *CartServiceImpl) RemoveItem(ctx *gin.Context, userID string, itemID uuid.UUID) error {
	if err := s.repo.RemoveCartItem(ctx, itemID); err != nil {
		return fmt.Errorf("failed to remove cart item: %w", err)
	}
	return nil
}

// ClearCart removes all items from the cart
func (s *CartServiceImpl) ClearCart(ctx *gin.Context, userID string) error {
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.repo.ClearCartItems(ctx, cart.ID); err != nil {
		return fmt.Errorf("failed to clear cart: %w", err)
	}

	return nil
}

// ConvertCart marks a cart as converted (e.g., when order is placed)
func (s *CartServiceImpl) ConvertCart(ctx *gin.Context, userID string) error {
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		return err
	}

	cart.Status = "converted"
	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		return fmt.Errorf("failed to convert cart: %w", err)
	}

	return nil
}

// SaveForLater moves an item from cart to saved items
func (s *CartServiceImpl) SaveForLater(ctx *gin.Context, userID string, itemID uuid.UUID) error {
	// Get cart item
	item, err := s.repo.GetCartItem(ctx, itemID)
	if err != nil {
		return err
	}

	// Create saved item
	savedItem := &domain.SavedItem{
		UserID:      userID,
		ProductID:   item.ProductID,
		ProductName: item.ProductName,
		ProductSKU:  item.ProductSKU,
		UnitPrice:   item.UnitPrice,
		Currency:    item.Currency,
		Variations:  item.Variations,
		Notes:       item.Notes,
	}

	// Save item
	if err := s.repo.SaveItemForLater(ctx, savedItem); err != nil {
		return fmt.Errorf("failed to save item for later: %w", err)
	}

	// Remove from cart
	return s.RemoveItem(ctx, userID, itemID)
}

// GetSavedItems gets all saved items for a user
func (s *CartServiceImpl) GetSavedItems(ctx *gin.Context, userID string) ([]domain.SavedItem, error) {
	return s.repo.GetSavedItems(ctx, userID)
}

// MoveToCart moves a saved item back to cart
func (s *CartServiceImpl) MoveToCart(ctx *gin.Context, userID string, savedItemID uuid.UUID) error {
	// This would involve getting the saved item and adding it back to cart
	// Implementation depends on your specific requirements
	return s.repo.MoveSavedItemToCart(ctx, userID, savedItemID)
}

// RemoveSavedItem removes a saved item
func (s *CartServiceImpl) RemoveSavedItem(ctx *gin.Context, userID string, savedItemID uuid.UUID) error {
	// Get saved item to find product ID
	savedItems, err := s.GetSavedItems(ctx, userID)
	if err != nil {
		return err
	}

	var productID uuid.UUID
	for _, item := range savedItems {
		if item.ID == savedItemID {
			productID = item.ProductID
			break
		}
	}

	if productID == uuid.Nil {
		return domain.ErrItemNotFound
	}

	return s.repo.RemoveSavedItem(ctx, userID, productID)
}

// MergeAnonymousCart merges an anonymous cart with a user's cart
func (s *CartServiceImpl) MergeAnonymousCart(ctx *gin.Context, sessionID string, userID string) (*domain.Cart, error) {
	// Get anonymous cart
	anonymousCart, err := s.repo.GetCartBySessionID(ctx, sessionID)
	if err != nil {
		// No anonymous cart to merge
		return s.GetOrCreateCart(ctx, userID, "")
	}

	// Get or create user cart
	userCart, err := s.GetOrCreateCart(ctx, userID, "")
	if err != nil {
		return nil, err
	}

	// Merge items from anonymous cart to user cart
	for _, item := range anonymousCart.Items {
		// Check if similar item exists in user cart
		existing := userCart.GetItemByProductID(item.ProductID, item.Variations)
		if existing != nil {
			// Merge quantities
			newQuantity := existing.Quantity + item.Quantity
			if _, err := s.UpdateItemQuantity(ctx, userID, existing.ID, newQuantity); err != nil {
				log.Printf("Failed to merge cart item quantity: %v", err)
			}
		} else {
			// Add new item to user cart
			item.CartID = userCart.ID
			item.ID = uuid.New() // Generate new ID
			if err := s.repo.AddItemToCart(ctx, userCart.ID, &item); err != nil {
				log.Printf("Failed to add cart item during merge: %v", err)
			}
		}
	}

	// Delete anonymous cart
	if err := s.repo.DeleteCart(ctx, anonymousCart.ID); err != nil {
		log.Printf("Failed to delete anonymous cart: %v", err)
	}

	// Return updated user cart
	return s.repo.GetCartWithItems(ctx, userCart.ID)
}

// GetAbandonedCarts gets carts that haven't been updated recently
func (s *CartServiceImpl) GetAbandonedCarts(ctx *gin.Context, since time.Time) ([]domain.Cart, error) {
	return s.repo.GetAbandonedCarts(ctx, &since)
}

// CleanupExpiredCarts removes expired carts
func (s *CartServiceImpl) CleanupExpiredCarts(ctx *gin.Context) (int64, error) {
	return s.repo.CleanupExpiredCarts(ctx)
}

// ValidateCartForCheckout validates cart before checkout
func (s *CartServiceImpl) ValidateCartForCheckout(ctx *gin.Context, userID string) (*domain.CartValidationResult, error) {
	cart, err := s.GetCart(ctx, userID)
	if err != nil {
		return &domain.CartValidationResult{
			IsValid: false,
			Errors:  []string{err.Error()},
		}, nil
	}

	result := &domain.CartValidationResult{
		IsValid:          true,
		Errors:           []string{},
		Warnings:         []string{},
		UnavailableItems: []uuid.UUID{},
		PriceChanges:     []domain.CartItemPriceChange{},
	}

	// Check if cart is empty
	if cart.IsEmpty() {
		result.IsValid = false
		result.Errors = append(result.Errors, "Cart is empty")
		return result, nil
	}

	// Check if cart is expired
	if cart.IsExpired() {
		result.IsValid = false
		result.Errors = append(result.Errors, "Cart has expired")
		return result, nil
	}

	// TODO: Validate product availability and prices
	// This would involve calling the product service to check current prices and stock

	return result, nil
}
