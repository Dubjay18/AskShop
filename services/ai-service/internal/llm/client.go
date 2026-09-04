// Package llm wraps the official Google Gen AI Go SDK (Gemini) behind a thin
// interface so the rest of ai-service depends on a small surface, not the SDK
// directly, and so the service can start (and degrade cleanly) without an API key.
package llm

import (
	"askshop/shared/env"
	"context"
	"log"

	"google.golang.org/genai"
)

// DefaultModel is used unless AI_MODEL overrides it. "flash" is the
// cost-appropriate default for a chat/search assistant; override via
// AI_MODEL (e.g. "gemini-3.1-pro-preview") for higher-reasoning workloads.
const DefaultModel = "gemini-3.6-flash"

// Client wraps the Gemini SDK client plus the model this service should use.
type Client struct {
	SDK       *genai.Client
	Model     string
	Available bool
}

// New builds a Client. It's always non-nil; Available is false when neither
// GEMINI_API_KEY nor GOOGLE_API_KEY is set, so callers can return a clear 503
// instead of making a doomed network call.
func New() *Client {
	apiKey := env.GetString("GEMINI_API_KEY", env.GetString("GOOGLE_API_KEY", ""))
	model := env.GetString("AI_MODEL", DefaultModel)

	client := &Client{Model: model, Available: apiKey != ""}
	if !client.Available {
		return client
	}

	sdk, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Printf("llm: failed to construct Gemini client, AI features disabled: %v", err)
		client.Available = false
		return client
	}

	client.SDK = sdk
	return client
}
