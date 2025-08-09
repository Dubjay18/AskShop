package rest

import (
	"askshop/shared/contracts"
	"askshop/shared/env"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ServiceClient is a REST client for service-to-service communication
type ServiceClient struct {
	baseURL     string
	httpClient  *http.Client
	serviceName string
}

// NewServiceClient creates a new service client
func NewServiceClient(serviceName string) *ServiceClient {
	baseURL := env.GetString(fmt.Sprintf("%s_SERVICE_URL", serviceName), fmt.Sprintf("http://%s-service:8080", serviceName))
	return &ServiceClient{
		baseURL:     baseURL,
		serviceName: serviceName,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Get performs a GET request to the specified path
func (c *ServiceClient) Get(ctx context.Context, path string, result interface{}) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, result)
}

// Post performs a POST request to the specified path with the given body
func (c *ServiceClient) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPost, path, body, result)
}

// Put performs a PUT request to the specified path with the given body
func (c *ServiceClient) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPut, path, body, result)
}

// Delete performs a DELETE request to the specified path
func (c *ServiceClient) Delete(ctx context.Context, path string, result interface{}) error {
	return c.doRequest(ctx, http.MethodDelete, path, nil, result)
}

// doRequest performs the actual HTTP request
func (c *ServiceClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("error marshaling request body: %w", err)
		}
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}
	}

	// Add request ID if present in context
	if reqID, ok := ctx.Value("request_id").(string); ok {
		req.Header.Set("X-Request-ID", reqID)
	}

	// Add authorization header if present in context
	if authHeader, ok := ctx.Value("authorization").(string); ok {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request to %s: %w", c.serviceName, err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	// Check if status code indicates error
	if resp.StatusCode >= 400 {
		var apiError contracts.APIResponse
		if err := json.Unmarshal(respBody, &apiError); err != nil {
			return fmt.Errorf("service returned status %d: %s", resp.StatusCode, string(respBody))
		}
		return fmt.Errorf("service returned error: %s", apiError.Error.Message)
	}

	// If no result is expected, return nil
	if result == nil {
		return nil
	}

	// Unmarshal response body into result
	var apiResp contracts.APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		// Try direct unmarshaling if APIResponse structure doesn't match
		return json.Unmarshal(respBody, result)
	}

	// If the response is in APIResponse format, unmarshal the data field
	if apiResp.Data != nil {
		jsonData, err := json.Marshal(apiResp.Data)
		if err != nil {
			return fmt.Errorf("error marshaling response data: %w", err)
		}
		return json.Unmarshal(jsonData, result)
	}

	return nil
}
