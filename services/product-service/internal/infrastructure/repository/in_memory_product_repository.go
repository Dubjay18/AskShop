package repository

import (
	"askshop/services/product-service/internal/domain"
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProductRepository defines the data access interface for product operations.
type ProductRepository interface {
	GetProductByID(ctx context.Context, productID string) (*domain.Product, error)
	GetProductByCategory(ctx context.Context, category string) ([]*domain.Product, error)
	GetAllProducts(ctx context.Context) ([]*domain.Product, error)
	// Paginated variants return items and total count
	GetAllProductsPage(ctx context.Context, limit, offset int) ([]*domain.Product, int64, error)
	GetProductByCategoryPage(ctx context.Context, category string, limit, offset int) ([]*domain.Product, int64, error)
	// SearchProducts does a keyword search over name/slug/sku (see domain.ScopeSearch).
	SearchProducts(ctx context.Context, query string, limit, offset int) ([]*domain.Product, int64, error)
	// Write operations
	CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error)
	UpdateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error)
	DeleteProduct(ctx context.Context, productID uuid.UUID) error
}

var ErrNotImplemented = errors.New("not implemented")
var ErrProductNotFound = errors.New("product not found")

// InMemoryProductRepository is a minimal, non-persistent fallback used when Postgres
// is unavailable. It's not category-aware and exists to keep the service usable in dev.
type InMemoryProductRepository struct {
	mu       sync.RWMutex
	products map[uuid.UUID]*domain.Product
}

func NewInMemoryProductRepository() *InMemoryProductRepository {
	return &InMemoryProductRepository{products: make(map[uuid.UUID]*domain.Product)}
}

func (r *InMemoryProductRepository) GetProductByID(ctx context.Context, productID string) (*domain.Product, error) {
	id, err := uuid.Parse(productID)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	if p, ok := r.products[id]; ok {
		r.mu.RUnlock()
		return p, nil
	}
	r.mu.RUnlock()

	// Minimal deterministic mock to enable cross-service integration when nothing
	// has been created yet (e.g. local dev without seeding).
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
		PriceCents:  2999,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (r *InMemoryProductRepository) GetProductByCategory(ctx context.Context, category string) ([]*domain.Product, error) {
	return []*domain.Product{}, nil
}

func (r *InMemoryProductRepository) GetAllProducts(ctx context.Context) ([]*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Product, 0, len(r.products))
	for _, p := range r.products {
		out = append(out, p)
	}
	return out, nil
}

func (r *InMemoryProductRepository) GetAllProductsPage(ctx context.Context, limit, offset int) ([]*domain.Product, int64, error) {
	all, _ := r.GetAllProducts(ctx)
	total := int64(len(all))
	if offset >= len(all) {
		return []*domain.Product{}, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (r *InMemoryProductRepository) GetProductByCategoryPage(ctx context.Context, category string, limit, offset int) ([]*domain.Product, int64, error) {
	return []*domain.Product{}, 0, nil
}

func (r *InMemoryProductRepository) SearchProducts(ctx context.Context, query string, limit, offset int) ([]*domain.Product, int64, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	all, _ := r.GetAllProducts(ctx)
	if q == "" {
		return all, int64(len(all)), nil
	}

	matched := make([]*domain.Product, 0, len(all))
	for _, p := range all {
		if strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.Slug), q) ||
			strings.Contains(strings.ToLower(p.SKU), q) {
			matched = append(matched, p)
		}
	}

	total := int64(len(matched))
	if offset >= len(matched) {
		return []*domain.Product{}, total, nil
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}
	return matched[offset:end], total, nil
}

func (r *InMemoryProductRepository) CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	if product.ID == uuid.Nil {
		product.ID = uuid.New()
	}
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now

	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[product.ID] = product
	return product, nil
}

func (r *InMemoryProductRepository) UpdateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[product.ID]; !ok {
		return nil, ErrProductNotFound
	}
	product.UpdatedAt = time.Now()
	r.products[product.ID] = product
	return product, nil
}

func (r *InMemoryProductRepository) DeleteProduct(ctx context.Context, productID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[productID]; !ok {
		return ErrProductNotFound
	}
	delete(r.products, productID)
	return nil
}
