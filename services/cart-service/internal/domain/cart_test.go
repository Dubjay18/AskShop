package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func TestCartItem_CalculateTotals(t *testing.T) {
	item := &CartItem{UnitPrice: 25, Quantity: 3, DiscountRate: 0.2}
	item.CalculateTotals()

	if item.Subtotal != 75 {
		t.Errorf("Subtotal = %v, want 75", item.Subtotal)
	}
	if item.DiscountTotal != 15 {
		t.Errorf("DiscountTotal = %v, want 15", item.DiscountTotal)
	}
	if item.FinalPrice != 60 {
		t.Errorf("FinalPrice = %v, want 60", item.FinalPrice)
	}
}

func TestCart_IsExpired(t *testing.T) {
	t.Run("nil expiry never expires", func(t *testing.T) {
		cart := &Cart{}
		if cart.IsExpired() {
			t.Error("cart with no expiry should not be expired")
		}
	})

	t.Run("future expiry not expired", func(t *testing.T) {
		future := time.Now().Add(time.Hour)
		cart := &Cart{ExpiresAt: &future}
		if cart.IsExpired() {
			t.Error("cart with future expiry should not be expired")
		}
	})

	t.Run("past expiry is expired", func(t *testing.T) {
		past := time.Now().Add(-time.Hour)
		cart := &Cart{ExpiresAt: &past}
		if !cart.IsExpired() {
			t.Error("cart with past expiry should be expired")
		}
	})
}

func TestCart_IsEmpty(t *testing.T) {
	if !(&Cart{}).IsEmpty() {
		t.Error("cart with no items should be empty")
	}
	if (&Cart{Items: []CartItem{{}}}).IsEmpty() {
		t.Error("cart with items should not be empty")
	}
}

func TestCart_GetItemByProductID(t *testing.T) {
	productID := uuid.New()
	otherID := uuid.New()

	cart := &Cart{Items: []CartItem{
		{ProductID: productID, Variations: datatypes.JSONMap{}},
		{ProductID: otherID, Variations: datatypes.JSONMap{"size": "L"}},
	}}

	t.Run("finds matching item with no variations", func(t *testing.T) {
		item := cart.GetItemByProductID(productID, datatypes.JSONMap{})
		if item == nil {
			t.Fatal("expected to find item, got nil")
		}
		if item.ProductID != productID {
			t.Errorf("ProductID = %v, want %v", item.ProductID, productID)
		}
	})

	t.Run("no match for unknown product", func(t *testing.T) {
		item := cart.GetItemByProductID(uuid.New(), datatypes.JSONMap{})
		if item != nil {
			t.Error("expected nil for unknown product ID")
		}
	})
}
