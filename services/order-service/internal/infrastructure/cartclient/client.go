// Package cartclient is a minimal HTTP client for order-service to read and
// finalize a user's cart in cart-service. It mirrors the request/response
// envelope used by shared/response.Standard.
package cartclient

import (
	"askshop/shared/env"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// CartItem mirrors the subset of cart-service's CartItem fields order-service needs.
type CartItem struct {
	ProductID   uuid.UUID `json:"productId"`
	ProductSKU  string    `json:"productSku"`
	ProductName string    `json:"productName"`
	UnitPrice   float64   `json:"unitPrice"`
	Quantity    int       `json:"quantity"`
}

// Cart mirrors the subset of cart-service's Cart fields order-service needs.
type Cart struct {
	ID       uuid.UUID  `json:"id"`
	Currency string     `json:"currency"`
	Items    []CartItem `json:"items"`
}

// ValidationResult mirrors cart-service's CartValidationResult.
type ValidationResult struct {
	IsValid bool     `json:"isValid"`
	Errors  []string `json:"errors"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New() *Client {
	baseURL := env.GetString("CART_SERVICE_URL", "http://cart-service:8083")
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) do(ctx context.Context, method, path, userID string, body interface{}, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("X-User-ID", userID)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("cart-service returned unexpected body (status %d): %s", resp.StatusCode, string(raw))
	}
	if !env.Success {
		msg := "cart-service request failed"
		if env.Error != nil && env.Error.Message != "" {
			msg = env.Error.Message
		}
		return fmt.Errorf("%s", msg)
	}
	if out != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

// GetCart fetches the user's current cart.
func (c *Client) GetCart(ctx context.Context, userID string) (*Cart, error) {
	var cart Cart
	if err := c.do(ctx, http.MethodGet, "/api/v1/cart", userID, nil, &cart); err != nil {
		return nil, err
	}
	return &cart, nil
}

// ValidateCheckout asks cart-service to validate the cart for checkout
// (stock and price checks against product-service).
func (c *Client) ValidateCheckout(ctx context.Context, userID string) (*ValidationResult, error) {
	var result ValidationResult
	if err := c.do(ctx, http.MethodPost, "/api/v1/cart/checkout/validate", userID, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CompleteCheckout marks the user's cart as converted after an order is created.
func (c *Client) CompleteCheckout(ctx context.Context, userID string) error {
	return c.do(ctx, http.MethodPost, "/api/v1/cart/checkout/complete", userID, nil, nil)
}
