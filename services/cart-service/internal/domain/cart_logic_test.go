package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func TestCartItemBuilder_Build(t *testing.T) {
	config := DefaultCartConfig()
	productID := uuid.New()

	t.Run("valid item", func(t *testing.T) {
		item, err := NewCartItemBuilder(config).
			WithProduct(productID, "SKU-1", "Widget", 19.99).
			WithQuantity(2).
			Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Subtotal != 39.98 {
			t.Errorf("Subtotal = %v, want 39.98", item.Subtotal)
		}
		if item.FinalPrice != 39.98 {
			t.Errorf("FinalPrice = %v, want 39.98", item.FinalPrice)
		}
	})

	t.Run("missing product id", func(t *testing.T) {
		_, err := NewCartItemBuilder(config).WithQuantity(1).Build()
		if err != ErrProductNotFound {
			t.Errorf("err = %v, want ErrProductNotFound", err)
		}
	})

	t.Run("zero quantity", func(t *testing.T) {
		_, err := NewCartItemBuilder(config).
			WithProduct(productID, "SKU-1", "Widget", 9.99).
			WithQuantity(0).
			Build()
		if err != ErrInvalidQuantity {
			t.Errorf("err = %v, want ErrInvalidQuantity", err)
		}
	})

	t.Run("quantity exceeds max", func(t *testing.T) {
		_, err := NewCartItemBuilder(config).
			WithProduct(productID, "SKU-1", "Widget", 9.99).
			WithQuantity(config.MaxQuantityPerItem + 1).
			Build()
		if err == nil {
			t.Error("expected error for quantity exceeding maximum, got nil")
		}
	})

	t.Run("negative price", func(t *testing.T) {
		_, err := NewCartItemBuilder(config).
			WithProduct(productID, "SKU-1", "Widget", -1).
			WithQuantity(1).
			Build()
		if err == nil {
			t.Error("expected error for negative price, got nil")
		}
	})

	t.Run("applies discount", func(t *testing.T) {
		item, err := NewCartItemBuilder(config).
			WithProduct(productID, "SKU-1", "Widget", 100).
			WithQuantity(1).
			WithDiscount(0.1).
			Build()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.DiscountTotal != 10 {
			t.Errorf("DiscountTotal = %v, want 10", item.DiscountTotal)
		}
		if item.FinalPrice != 90 {
			t.Errorf("FinalPrice = %v, want 90", item.FinalPrice)
		}
	})
}

func TestCartManager_ValidateAddItem(t *testing.T) {
	config := DefaultCartConfig()
	config.MaxItemsPerCart = 2
	cm := NewCartManager(config)
	productID := uuid.New()

	t.Run("full cart rejected", func(t *testing.T) {
		cart := &Cart{Items: []CartItem{{}, {}}}
		item := &CartItem{ProductID: uuid.New()}
		if err := cm.ValidateAddItem(cart, item); err == nil {
			t.Error("expected error for full cart, got nil")
		}
	})

	t.Run("expired cart rejected", func(t *testing.T) {
		past := time.Now().Add(-time.Hour)
		cart := &Cart{ExpiresAt: &past}
		item := &CartItem{ProductID: uuid.New()}
		if err := cm.ValidateAddItem(cart, item); err != ErrCartExpired {
			t.Errorf("err = %v, want ErrCartExpired", err)
		}
	})

	t.Run("duplicate item rejected", func(t *testing.T) {
		cart := &Cart{Items: []CartItem{{ProductID: productID}}}
		item := &CartItem{ProductID: productID}
		if err := cm.ValidateAddItem(cart, item); err != ErrDuplicateItem {
			t.Errorf("err = %v, want ErrDuplicateItem", err)
		}
	})

	t.Run("valid add", func(t *testing.T) {
		cart := &Cart{}
		item := &CartItem{ProductID: uuid.New()}
		if err := cm.ValidateAddItem(cart, item); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCartManager_ValidateQuantityUpdate(t *testing.T) {
	cm := NewCartManager(DefaultCartConfig())

	if err := cm.ValidateQuantityUpdate(-1); err != ErrInvalidQuantity {
		t.Errorf("err = %v, want ErrInvalidQuantity", err)
	}
	if err := cm.ValidateQuantityUpdate(1000); err == nil {
		t.Error("expected error for quantity exceeding maximum, got nil")
	}
	if err := cm.ValidateQuantityUpdate(5); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCartManager_CalculateCartSummary(t *testing.T) {
	config := DefaultCartConfig()
	config.TaxRate = 0.1
	config.FreeShippingMinimum = 100
	config.StandardShippingFee = 5
	cm := NewCartManager(config)

	t.Run("below free shipping threshold", func(t *testing.T) {
		cart := &Cart{
			Items: []CartItem{
				{UnitPrice: 10, Quantity: 2}, // subtotal 20
			},
		}
		summary := cm.CalculateCartSummary(cart)
		if summary.Subtotal != 20 {
			t.Errorf("Subtotal = %v, want 20", summary.Subtotal)
		}
		if summary.TaxTotal != 2 {
			t.Errorf("TaxTotal = %v, want 2", summary.TaxTotal)
		}
		if summary.ShippingTotal != 5 {
			t.Errorf("ShippingTotal = %v, want 5", summary.ShippingTotal)
		}
		if summary.GrandTotal != 27 {
			t.Errorf("GrandTotal = %v, want 27", summary.GrandTotal)
		}
	})

	t.Run("meets free shipping threshold", func(t *testing.T) {
		cart := &Cart{
			Items: []CartItem{
				{UnitPrice: 100, Quantity: 1},
			},
		}
		summary := cm.CalculateCartSummary(cart)
		if summary.ShippingTotal != 0 {
			t.Errorf("ShippingTotal = %v, want 0", summary.ShippingTotal)
		}
	})
}

func TestCartManager_ShouldMergeItems(t *testing.T) {
	cm := NewCartManager(DefaultCartConfig())
	productID := uuid.New()

	same := datatypes.JSONMap{"color": "blue"}
	a := &CartItem{ProductID: productID, ProductSKU: "SKU-1", Variations: same}
	b := &CartItem{ProductID: productID, ProductSKU: "SKU-1", Variations: datatypes.JSONMap{"color": "blue"}}
	if !cm.ShouldMergeItems(a, b) {
		t.Error("expected items with matching product/SKU/variations to merge")
	}

	c := &CartItem{ProductID: productID, ProductSKU: "SKU-1", Variations: datatypes.JSONMap{"color": "red"}}
	if cm.ShouldMergeItems(a, c) {
		t.Error("expected items with different variations not to merge")
	}
}

func TestCartManager_IsAbandonedCart(t *testing.T) {
	cm := NewCartManager(DefaultCartConfig())
	threshold := time.Hour

	t.Run("empty cart is never abandoned", func(t *testing.T) {
		cart := &Cart{Status: "active", UpdatedAt: time.Now().Add(-2 * time.Hour)}
		if cm.IsAbandonedCart(cart, threshold) {
			t.Error("empty cart should not be considered abandoned")
		}
	})

	t.Run("recently updated cart is not abandoned", func(t *testing.T) {
		cart := &Cart{
			Status:    "active",
			UpdatedAt: time.Now(),
			Items:     []CartItem{{ID: uuid.New()}},
		}
		if cm.IsAbandonedCart(cart, threshold) {
			t.Error("recently updated cart should not be considered abandoned")
		}
	})

	t.Run("stale active cart is abandoned", func(t *testing.T) {
		cart := &Cart{
			Status:    "active",
			UpdatedAt: time.Now().Add(-2 * time.Hour),
			Items:     []CartItem{{ID: uuid.New()}},
		}
		if !cm.IsAbandonedCart(cart, threshold) {
			t.Error("stale cart with items should be considered abandoned")
		}
	})

	t.Run("converted cart is never abandoned", func(t *testing.T) {
		cart := &Cart{
			Status:    "converted",
			UpdatedAt: time.Now().Add(-2 * time.Hour),
			Items:     []CartItem{{ID: uuid.New()}},
		}
		if cm.IsAbandonedCart(cart, threshold) {
			t.Error("converted cart should not be considered abandoned")
		}
	})
}
