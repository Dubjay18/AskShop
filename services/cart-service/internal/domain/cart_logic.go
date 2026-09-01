package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Cart business logic and validation

var (
	ErrCartNotFound      = errors.New("cart not found")
	ErrItemNotFound      = errors.New("cart item not found")
	ErrInvalidQuantity   = errors.New("invalid quantity")
	ErrProductNotFound   = errors.New("product not found")
	ErrCartExpired       = errors.New("cart has expired")
	ErrEmptyCart         = errors.New("cart is empty")
	ErrItemNotInCart     = errors.New("item not in cart")
	ErrDuplicateItem     = errors.New("item already exists in cart")
	ErrInsufficientStock = errors.New("insufficient stock")
)

// CartConfig holds configuration for cart behavior
type CartConfig struct {
	MaxItemsPerCart     int           `json:"maxItemsPerCart"`
	MaxQuantityPerItem  int           `json:"maxQuantityPerItem"`
	DefaultCartExpiry   time.Duration `json:"defaultCartExpiry"`
	AllowGuestCheckout  bool          `json:"allowGuestCheckout"`
	Currency            string        `json:"currency"`
	TaxRate             float64       `json:"taxRate"`
	FreeShippingMinimum float64       `json:"freeShippingMinimum"`
	StandardShippingFee float64       `json:"standardShippingFee"`
}

// DefaultCartConfig returns default configuration
func DefaultCartConfig() CartConfig {
	return CartConfig{
		MaxItemsPerCart:     50,
		MaxQuantityPerItem:  99,
		DefaultCartExpiry:   30 * 24 * time.Hour, // 30 days
		AllowGuestCheckout:  true,
		Currency:            "USD",
		TaxRate:             0.08, // 8%
		FreeShippingMinimum: 100.0,
		StandardShippingFee: 9.99,
	}
}

// CartItemBuilder helps build cart items with validation
type CartItemBuilder struct {
	item   CartItem
	config CartConfig
}

// NewCartItemBuilder creates a new cart item builder
func NewCartItemBuilder(config CartConfig) *CartItemBuilder {
	return &CartItemBuilder{
		config: config,
		item: CartItem{
			Currency: config.Currency,
			Quantity: 1,
		},
	}
}

// WithProduct sets the product information
func (b *CartItemBuilder) WithProduct(productID uuid.UUID, sku, name string, price float64) *CartItemBuilder {
	b.item.ProductID = productID
	b.item.ProductSKU = sku
	b.item.ProductName = name
	b.item.UnitPrice = price
	return b
}

// WithQuantity sets the quantity
func (b *CartItemBuilder) WithQuantity(quantity int) *CartItemBuilder {
	b.item.Quantity = quantity
	return b
}

// WithVariations sets the product variations
func (b *CartItemBuilder) WithVariations(variations datatypes.JSONMap) *CartItemBuilder {
	b.item.Variations = variations
	return b
}

// WithDiscount sets the discount rate
func (b *CartItemBuilder) WithDiscount(discountRate float64) *CartItemBuilder {
	b.item.DiscountRate = discountRate
	return b
}

// WithNotes sets the item notes
func (b *CartItemBuilder) WithNotes(notes string) *CartItemBuilder {
	b.item.Notes = notes
	return b
}

// Build creates and validates the cart item
func (b *CartItemBuilder) Build() (*CartItem, error) {
	// Validation
	if b.item.ProductID == uuid.Nil {
		return nil, ErrProductNotFound
	}
	if b.item.Quantity <= 0 {
		return nil, ErrInvalidQuantity
	}
	if b.item.Quantity > b.config.MaxQuantityPerItem {
		return nil, errors.New("quantity exceeds maximum allowed")
	}
	if b.item.UnitPrice < 0 {
		return nil, errors.New("invalid price")
	}

	// Calculate totals
	b.item.CalculateTotals()

	return &b.item, nil
}

// CartManager provides cart business logic operations
type CartManager struct {
	config CartConfig
}

// NewCartManager creates a new cart manager
func NewCartManager(config CartConfig) *CartManager {
	return &CartManager{config: config}
}

// ValidateAddItem validates adding an item to cart
func (cm *CartManager) ValidateAddItem(cart *Cart, item *CartItem) error {
	// Check cart capacity
	if len(cart.Items) >= cm.config.MaxItemsPerCart {
		return errors.New("cart is full")
	}

	// Check if cart is expired
	if cart.IsExpired() {
		return ErrCartExpired
	}

	// Check for duplicate items with same variations
	existing := cart.GetItemByProductID(item.ProductID, item.Variations)
	if existing != nil {
		return ErrDuplicateItem
	}

	return nil
}

// ValidateQuantityUpdate validates updating item quantity
func (cm *CartManager) ValidateQuantityUpdate(quantity int) error {
	if quantity < 0 {
		return ErrInvalidQuantity
	}
	if quantity > cm.config.MaxQuantityPerItem {
		return errors.New("quantity exceeds maximum allowed")
	}
	return nil
}

// CalculateCartSummary calculates the cart summary with taxes and shipping
func (cm *CartManager) CalculateCartSummary(cart *Cart) CartSummary {
	summary := cart.CalculateSummary()

	// Apply configuration-based calculations
	summary.TaxTotal = summary.Subtotal * cm.config.TaxRate

	// Shipping calculation
	if summary.Subtotal >= cm.config.FreeShippingMinimum {
		summary.ShippingTotal = 0
	} else {
		summary.ShippingTotal = cm.config.StandardShippingFee
	}

	// Recalculate grand total
	summary.GrandTotal = summary.Subtotal - summary.DiscountTotal + summary.TaxTotal + summary.ShippingTotal

	return summary
}

// ShouldMergeItems determines if two cart items should be merged
func (cm *CartManager) ShouldMergeItems(existing, new *CartItem) bool {
	return existing.ProductID == new.ProductID &&
		existing.ProductSKU == new.ProductSKU &&
		variationsEqual(existing.Variations, new.Variations)
}

// variationsEqual compares two variation maps
func variationsEqual(v1, v2 datatypes.JSONMap) bool {
	if len(v1) != len(v2) {
		return false
	}
	for k, val1 := range v1 {
		if val2, exists := v2[k]; !exists || val1 != val2 {
			return false
		}
	}
	return true
}

// IsAbandonedCart determines if a cart is considered abandoned
func (cm *CartManager) IsAbandonedCart(cart *Cart, abandonedThreshold time.Duration) bool {
	return !cart.IsEmpty() &&
		cart.Status == "active" &&
		time.Since(cart.UpdatedAt) > abandonedThreshold
}

// CartStatistics provides cart analytics data
type CartStatistics struct {
	TotalCarts     int64   `json:"totalCarts"`
	ActiveCarts    int64   `json:"activeCarts"`
	AbandonedCarts int64   `json:"abandonedCarts"`
	ConvertedCarts int64   `json:"convertedCarts"`
	AverageValue   float64 `json:"averageValue"`
	AverageItems   float64 `json:"averageItems"`
	ConversionRate float64 `json:"conversionRate"`
}
