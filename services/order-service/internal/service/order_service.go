package service

import (
	"askshop/services/order-service/internal/domain"
	"askshop/services/order-service/internal/infrastructure/cartclient"
	"askshop/shared/contracts"
	"askshop/shared/events"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

// OrderServiceImpl implements domain.OrderService, coordinating with cart-service
// over HTTP and publishing order lifecycle events over RabbitMQ.
type OrderServiceImpl struct {
	repo      domain.OrderRepository
	cartCli   *cartclient.Client
	publisher *events.Publisher
}

func NewOrderService(repo domain.OrderRepository, cartCli *cartclient.Client, publisher *events.Publisher) domain.OrderService {
	return &OrderServiceImpl{repo: repo, cartCli: cartCli, publisher: publisher}
}

// PlaceOrder validates the user's cart, snapshots it into an Order, marks the
// cart as converted, and publishes an order.event.placed event.
func (s *OrderServiceImpl) PlaceOrder(ctx context.Context, userID string) (*domain.Order, error) {
	validation, err := s.cartCli.ValidateCheckout(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate cart: %w", err)
	}
	if !validation.IsValid {
		return nil, fmt.Errorf("%w: %s", domain.ErrCartInvalid, strings.Join(validation.Errors, "; "))
	}

	cart, err := s.cartCli.GetCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load cart: %w", err)
	}
	if len(cart.Items) == 0 {
		return nil, domain.ErrCartEmpty
	}

	order := &domain.Order{
		ID:       uuid.New(),
		UserID:   userID,
		Status:   domain.StatusPlaced,
		Currency: cart.Currency,
	}
	for _, item := range cart.Items {
		unitPriceCents := int64(item.UnitPrice*100 + 0.5)
		order.Items = append(order.Items, domain.OrderItem{
			ProductID:      item.ProductID,
			ProductSKU:     item.ProductSKU,
			ProductName:    item.ProductName,
			UnitPriceCents: unitPriceCents,
			Quantity:       item.Quantity,
		})
		order.SubtotalCents += unitPriceCents * int64(item.Quantity)
	}
	order.TotalCents = order.SubtotalCents

	created, err := s.repo.CreateOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	if err := s.cartCli.CompleteCheckout(ctx, userID); err != nil {
		// The order is already durably created; log and continue rather than
		// failing the whole checkout over cart bookkeeping.
		log.Printf("order-service: failed to mark cart converted for user %s: %v", userID, err)
	}

	if err := s.publisher.Publish(ctx, contracts.OrderEventPlaced, created); err != nil {
		log.Printf("order-service: failed to publish %s for order %s: %v", contracts.OrderEventPlaced, created.ID, err)
	}

	return created, nil
}

func (s *OrderServiceImpl) GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	return s.repo.GetOrderByID(ctx, id)
}

func (s *OrderServiceImpl) GetOrdersForUser(ctx context.Context, userID string) ([]*domain.Order, error) {
	return s.repo.GetOrdersByUserID(ctx, userID)
}
