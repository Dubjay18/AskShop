package repository

import (
	"askshop/services/product-service/internal/domain"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ensure PGProductRepository satisfies ProductRepository at compile time.
var _ ProductRepository = (*PGProductRepository)(nil)

// PGProductRepository implements ProductRepository backed by Postgres via GORM.
type PGProductRepository struct {
	db *gorm.DB
}

// NewPGProductRepository creates a new Postgres-backed product repository.
func NewPGProductRepository(db *gorm.DB) *PGProductRepository {
	return &PGProductRepository{db: db}
}

func (r *PGProductRepository) preload() *gorm.DB {
	return r.db.Preload("Images").Preload("Categories")
}

func (r *PGProductRepository) GetProductByID(ctx context.Context, productID string) (*domain.Product, error) {
	id, err := uuid.Parse(productID)
	if err != nil {
		return nil, err
	}
	var product domain.Product
	if err := r.preload().WithContext(ctx).First(&product, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *PGProductRepository) GetProductByCategory(ctx context.Context, category string) ([]*domain.Product, error) {
	var products []*domain.Product
	err := r.preload().WithContext(ctx).
		Joins("JOIN product_categories ON product_categories.product_id = products.id").
		Joins("JOIN categories ON categories.id = product_categories.category_id").
		Where("categories.slug = ?", category).
		Find(&products).Error
	return products, err
}

func (r *PGProductRepository) GetAllProducts(ctx context.Context) ([]*domain.Product, error) {
	var products []*domain.Product
	err := r.preload().WithContext(ctx).Find(&products).Error
	return products, err
}

func (r *PGProductRepository) GetAllProductsPage(ctx context.Context, limit, offset int) ([]*domain.Product, int64, error) {
	var products []*domain.Product
	var total int64
	if err := r.db.WithContext(ctx).Model(&domain.Product{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.preload().WithContext(ctx).Limit(limit).Offset(offset).Order("created_at DESC").Find(&products).Error
	return products, total, err
}

func (r *PGProductRepository) GetProductByCategoryPage(ctx context.Context, category string, limit, offset int) ([]*domain.Product, int64, error) {
	base := r.db.WithContext(ctx).Model(&domain.Product{}).
		Joins("JOIN product_categories ON product_categories.product_id = products.id").
		Joins("JOIN categories ON categories.id = product_categories.category_id").
		Where("categories.slug = ?", category)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var products []*domain.Product
	err := r.preload().WithContext(ctx).
		Joins("JOIN product_categories ON product_categories.product_id = products.id").
		Joins("JOIN categories ON categories.id = product_categories.category_id").
		Where("categories.slug = ?", category).
		Limit(limit).Offset(offset).Order("products.created_at DESC").
		Find(&products).Error
	return products, total, err
}

func (r *PGProductRepository) CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	if err := r.db.WithContext(ctx).Create(product).Error; err != nil {
		return nil, err
	}
	return product, nil
}

func (r *PGProductRepository) UpdateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	if err := r.db.WithContext(ctx).Model(&domain.Product{}).Where("id = ?", product.ID).Updates(product).Error; err != nil {
		return nil, err
	}
	return r.GetProductByID(ctx, product.ID.String())
}

func (r *PGProductRepository) DeleteProduct(ctx context.Context, productID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Product{}, "id = ?", productID).Error
}
