// Package productclient is a minimal HTTP client for ai-service to search and
// fetch products from product-service, so the shopping assistant only ever
// talks about products that actually exist in the catalog.
package productclient

import (
	"askshop/shared/env"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// StatusError preserves product-service's HTTP status so callers can map
// "not found" (404) and "bad request" (400) distinctly instead of collapsing
// every failure to 500.
type StatusError struct {
	StatusCode int
	Message    string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("product-service returned %d: %s", e.StatusCode, e.Message)
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Product mirrors the subset of product-service's ProductResponse fields
// ai-service needs to ground its answers in real catalog data.
type Product struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	SKU           string   `json:"sku"`
	Description   string   `json:"description"`
	Currency      string   `json:"currency"`
	Status        string   `json:"status"`
	Tags          []string `json:"tags"`
	PriceCents    int64    `json:"priceCents"`
	StockQuantity int      `json:"stockQuantity"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New() *Client {
	baseURL := env.GetString("PRODUCT_SERVICE_URL", "http://product-service:8082")
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
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
		return &StatusError{StatusCode: resp.StatusCode, Message: string(raw)}
	}
	if !env.Success {
		msg := "product-service request failed"
		if env.Error != nil && env.Error.Message != "" {
			msg = env.Error.Message
		}
		return &StatusError{StatusCode: resp.StatusCode, Message: msg}
	}
	if out != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

// Search does a keyword search over the catalog, capped at limit results.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Product, error) {
	if limit <= 0 {
		limit = 10
	}
	var products []Product
	path := "/api/v1/products?" + url.Values{
		"q":        {query},
		"pageSize": {fmt.Sprintf("%d", limit)},
	}.Encode()
	if err := c.get(ctx, path, &products); err != nil {
		return nil, err
	}
	return products, nil
}

// GetByID fetches a single product by id.
func (c *Client) GetByID(ctx context.Context, id string) (*Product, error) {
	var p Product
	if err := c.get(ctx, "/api/v1/products/"+id, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
