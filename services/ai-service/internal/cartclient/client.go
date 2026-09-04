// Package cartclient is a minimal HTTP client for ai-service to read
// abandoned carts from cart-service's admin endpoint, used to generate
// cart-abandonment nudges.
package cartclient

import (
	"askshop/shared/env"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// CartItem mirrors the subset of cart-service's CartItem fields needed for nudges.
type CartItem struct {
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	UnitPrice   float64 `json:"unitPrice"`
	Quantity    int     `json:"quantity"`
}

// Cart mirrors the subset of cart-service's Cart fields needed for nudges.
type Cart struct {
	ID       string     `json:"id"`
	UserID   string     `json:"userId"`
	Currency string     `json:"currency"`
	Items    []CartItem `json:"items"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New() *Client {
	baseURL := env.GetString("CART_SERVICE_URL", "http://cart-service:8083")
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

// ListAbandoned returns carts untouched for at least sinceMinutes.
func (c *Client) ListAbandoned(ctx context.Context, sinceMinutes int) ([]Cart, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1/cart/admin/abandoned?sinceMinutes=%d", c.baseURL, sinceMinutes), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("cart-service returned unexpected body (status %d): %s", resp.StatusCode, string(raw))
	}
	if !env.Success {
		msg := "cart-service request failed"
		if env.Error != nil && env.Error.Message != "" {
			msg = env.Error.Message
		}
		return nil, fmt.Errorf("%s", msg)
	}

	var carts []Cart
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &carts); err != nil {
			return nil, err
		}
	}
	return carts, nil
}
