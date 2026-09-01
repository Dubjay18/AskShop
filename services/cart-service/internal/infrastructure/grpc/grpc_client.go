package productclient

import (
	"askshop/shared/proto"
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ProductInfo is the subset of product data cart-service needs for pricing and
// availability checks.
type ProductInfo struct {
	ID            string
	Name          string
	SKU           string
	Status        string
	PriceCents    int64
	StockQuantity int32
}

// ErrProductUnavailable is returned when a product can't be sold (missing or not active).
var ErrProductUnavailable = errors.New("product unavailable")

// Client defines minimal product lookups over gRPC.
type Client interface {
	// GetProductSKU returns the SKU for a product ID.
	GetProductSKU(ctx context.Context, productID string) (string, error)
	// GetProduct returns pricing/availability info for a product ID.
	GetProduct(ctx context.Context, productID string) (*ProductInfo, error)
}

type GRPCClient struct {
	addr        string
	dialTimeout time.Duration
}

// NewGRPCClient creates a new product gRPC client.
func NewGRPCClient(addr string) *GRPCClient {
	return &GRPCClient{addr: addr, dialTimeout: 3 * time.Second}
}

func (c *GRPCClient) dial(ctx context.Context) (*grpc.ClientConn, error) {
	dctx, cancel := context.WithTimeout(ctx, c.dialTimeout)
	defer cancel()

	return grpc.DialContext(
		dctx,
		c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}

// GetProductSKU dials the product service and fetches product SKU via gRPC.
func (c *GRPCClient) GetProductSKU(ctx context.Context, productID string) (string, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	client := proto.NewProductServiceClient(conn)
	req := &proto.ProductRequest{Identifier: &proto.ProductRequest_Id{Id: productID}}
	resp, err := client.GetProduct(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.GetSku(), nil
}

// GetProduct dials the product service and fetches pricing/availability info via gRPC.
func (c *GRPCClient) GetProduct(ctx context.Context, productID string) (*ProductInfo, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := proto.NewProductServiceClient(conn)
	req := &proto.ProductRequest{Identifier: &proto.ProductRequest_Id{Id: productID}}
	resp, err := client.GetProduct(ctx, req)
	if err != nil {
		return nil, err
	}

	info := &ProductInfo{
		ID:            resp.GetId(),
		Name:          resp.GetName(),
		SKU:           resp.GetSku(),
		Status:        resp.GetStatus(),
		PriceCents:    resp.GetPriceCents(),
		StockQuantity: resp.GetStockQuantity(),
	}
	if info.Status != "active" {
		return info, ErrProductUnavailable
	}
	return info, nil
}
