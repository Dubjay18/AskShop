package repository

import (
	"askshop/services/order-service/internal/domain"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var _ domain.OrderRepository = (*PGOrderRepository)(nil)

type PGOrderRepository struct {
	db *gorm.DB
}

func NewPGOrderRepository(db *gorm.DB) *PGOrderRepository {
	return &PGOrderRepository{db: db}
}

func (r *PGOrderRepository) CreateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	if err := r.db.WithContext(ctx).Create(order).Error; err != nil {
		return nil, err
	}
	return order, nil
}

func (r *PGOrderRepository) GetOrderByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.WithContext(ctx).Preload("Items").First(&order, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *PGOrderRepository) GetOrdersByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	var orders []*domain.Order
	err := r.db.WithContext(ctx).Preload("Items").Where("user_id = ?", userID).Order("created_at DESC").Find(&orders).Error
	return orders, err
}

func (r *PGOrderRepository) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&domain.Order{}).Where("id = ?", id).Update("status", status).Error
}
