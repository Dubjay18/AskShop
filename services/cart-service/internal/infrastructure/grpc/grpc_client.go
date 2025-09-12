package productclient

import (
	"askshop/shared/proto"
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client defines minimal product lookups over gRPC.
type Client interface {
	// GetProductSKU returns the SKU for a product ID.
	GetProductSKU(ctx context.Context, productID string) (string, error)
}

type GRPCClient struct {
	addr        string
	dialTimeout time.Duration
}

// NewGRPCClient creates a new product gRPC client.
func NewGRPCClient(addr string) *GRPCClient {
	return &GRPCClient{addr: addr, dialTimeout: 3 * time.Second}
}

// GetProductSKU dials the product service and fetches product SKU via gRPC.
func (c *GRPCClient) GetProductSKU(ctx context.Context, productID string) (string, error) {
	dctx, cancel := context.WithTimeout(ctx, c.dialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(
		dctx,
		c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	client := proto.NewProductServiceClient(conn)
	// Build a request by ID; adjust if using slug or sku.
	req := &proto.ProductRequest{Identifier: &proto.ProductRequest_Id{Id: productID}}
	resp, err := client.GetProduct(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.GetSku(), nil
}
