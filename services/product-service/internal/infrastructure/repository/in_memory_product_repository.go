package repository

import (
    "askshop/services/product-service/internal/domain"
    "context"
    "errors"
    "time"

    "github.com/google/uuid"
)

type InMemoryProductRepository struct{}

func NewInMemoryProductRepository() *InMemoryProductRepository { return &InMemoryProductRepository{} }

func (r *InMemoryProductRepository) GetProductByID(ctx context.Context, productID string) (*domain.Product, error) {
    id, err := uuid.Parse(productID)
    if err != nil {
        return nil, err
    }
    // Minimal deterministic mock to enable cross-service integration
    now := time.Now()
    sku := "SKU-" + id.String()[:8]
    return &domain.Product{
        ID:          id,
        Name:        "Product " + id.String()[:8],
        Slug:        id.String(),
        SKU:         sku,
        Description: "",
        Currency:    "USD",
        Status:      "active",
        CreatedAt:   now,
        UpdatedAt:   now,
    }, nil
}

func (r *InMemoryProductRepository) GetProductByCategory(ctx context.Context, category string) ([]*domain.Product, error) {
    return []*domain.Product{}, nil
}

func (r *InMemoryProductRepository) GetAllProducts(ctx context.Context) ([]*domain.Product, error) {
    return []*domain.Product{}, nil
}

func (r *InMemoryProductRepository) GetAllProductsPage(ctx context.Context, limit, offset int) ([]*domain.Product, int64, error) {
    return []*domain.Product{}, 0, nil
}

func (r *InMemoryProductRepository) GetProductByCategoryPage(ctx context.Context, category string, limit, offset int) ([]*domain.Product, int64, error) {
    return []*domain.Product{}, 0, nil
}

var ErrNotImplemented = errors.New("not implemented")
