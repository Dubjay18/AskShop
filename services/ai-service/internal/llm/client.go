// Package llm wraps the official Anthropic Go SDK behind a thin interface so
// the rest of ai-service depends on a small surface, not the SDK directly,
// and so the service can start (and degrade cleanly) without an API key.
package llm

import (
	"askshop/shared/env"

	"github.com/anthropics/anthropic-sdk-go"
)

// DefaultModel is used unless AI_MODEL overrides it. Claude Opus 5 is the
// current default per Anthropic's guidance; override via AI_MODEL (e.g.
// "claude-sonnet-5" or "claude-haiku-4-5") for a lower-cost deployment.
const DefaultModel = "claude-opus-5"

// Client wraps the Anthropic SDK client plus the model this service should use.
type Client struct {
	SDK       anthropic.Client
	Model     string
	Available bool
}

// New builds a Client. It's always non-nil; Available is false when
// ANTHROPIC_API_KEY is unset so callers can return a clear 503 instead of
// making a doomed network call.
func New() *Client {
	apiKey := env.GetString("ANTHROPIC_API_KEY", "")
	model := env.GetString("AI_MODEL", DefaultModel)
	return &Client{
		SDK:       anthropic.NewClient(), // reads ANTHROPIC_API_KEY itself
		Model:     model,
		Available: apiKey != "",
	}
}
