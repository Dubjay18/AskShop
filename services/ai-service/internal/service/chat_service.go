// Package service implements ai-service's business logic: a conversational
// shopping assistant grounded in the real product catalog via tool use, plus
// single-shot product explanations and cart-abandonment nudges.
package service

import (
	"askshop/services/ai-service/internal/cartclient"
	"askshop/services/ai-service/internal/llm"
	"askshop/services/ai-service/internal/productclient"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

var ErrLLMUnavailable = errors.New("AI features are unavailable: ANTHROPIC_API_KEY is not configured")

const maxToolIterations = 4

type ChatMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

type ChatResult struct {
	Reply          string   `json:"reply"`
	ReferencedSKUs []string `json:"referencedSkus,omitempty"`
}

// AIService coordinates the LLM with product/cart data so answers stay
// grounded in what's actually in the catalog rather than the model guessing.
type AIService struct {
	llm     *llm.Client
	product *productclient.Client
	cart    *cartclient.Client
}

func NewAIService(llmClient *llm.Client, productClient *productclient.Client, cartClient *cartclient.Client) *AIService {
	return &AIService{llm: llmClient, product: productClient, cart: cartClient}
}

const chatSystemPrompt = `You are AskShop's shopping assistant. Help customers find products in
the catalog and answer questions about them. Always use the search_products tool to look up
products before recommending or describing anything specific — never invent products, prices,
or stock levels. If nothing in the catalog matches, say so plainly. Keep replies short (2-4
sentences) and mention prices in dollars (priceCents / 100), formatted like $19.99.`

var searchProductsTool = anthropic.ToolParam{
	Name:        "search_products",
	Description: anthropic.String("Search the AskShop product catalog by keyword. Returns matching products with id, name, price, and stock."),
	InputSchema: anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Keywords to search for, e.g. 'wireless headphones'",
			},
		},
		Required: []string{"query"},
	},
}

// Chat answers a user's shopping question, using search_products as a tool
// so the model can ground its reply in the real catalog.
func (s *AIService) Chat(ctx context.Context, history []ChatMessage, userMessage string) (*ChatResult, error) {
	if !s.llm.Available {
		return nil, ErrLLMUnavailable
	}

	messages := make([]anthropic.MessageParam, 0, len(history)+1)
	for _, m := range history {
		if m.Role == "assistant" {
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		} else {
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		}
	}
	messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)))

	tools := []anthropic.ToolUnionParam{{OfTool: &searchProductsTool}}
	referenced := map[string]struct{}{}

	var finalText strings.Builder
	for i := 0; i < maxToolIterations; i++ {
		resp, err := s.llm.SDK.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(s.llm.Model),
			MaxTokens: 1024,
			System:    []anthropic.TextBlockParam{{Text: chatSystemPrompt}},
			Messages:  messages,
			Tools:     tools,
		})
		if err != nil {
			return nil, fmt.Errorf("assistant call failed: %w", err)
		}

		messages = append(messages, resp.ToParam())

		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, block := range resp.Content {
			switch variant := block.AsAny().(type) {
			case anthropic.TextBlock:
				finalText.WriteString(variant.Text)
			case anthropic.ToolUseBlock:
				result, skus := s.runTool(ctx, variant.Name, variant.JSON.Input.Raw())
				for _, sku := range skus {
					referenced[sku] = struct{}{}
				}
				toolResults = append(toolResults, anthropic.NewToolResultBlock(variant.ID, result, false))
			}
		}

		if resp.StopReason != anthropic.StopReasonToolUse {
			break
		}
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
		finalText.Reset() // only the post-tool-use reply matters
	}

	skus := make([]string, 0, len(referenced))
	for sku := range referenced {
		skus = append(skus, sku)
	}

	reply := strings.TrimSpace(finalText.String())
	if reply == "" {
		reply = "I couldn't find anything matching that in our catalog right now."
	}
	return &ChatResult{Reply: reply, ReferencedSKUs: skus}, nil
}

// runTool executes a tool call by name and returns its result text plus any
// product SKUs it surfaced (for the caller to track what was referenced).
func (s *AIService) runTool(ctx context.Context, name, rawInput string) (string, []string) {
	switch name {
	case "search_products":
		var in struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(rawInput), &in); err != nil {
			return fmt.Sprintf("invalid tool input: %v", err), nil
		}
		products, err := s.product.Search(ctx, in.Query, 5)
		if err != nil {
			log.Printf("ai-service: search_products failed: %v", err)
			return fmt.Sprintf("search failed: %v", err), nil
		}
		if len(products) == 0 {
			return "no matching products found", nil
		}

		var sb strings.Builder
		skus := make([]string, 0, len(products))
		for _, p := range products {
			fmt.Fprintf(&sb, "- %s (sku=%s, id=%s): $%.2f, %d in stock, status=%s\n",
				p.Name, p.SKU, p.ID, float64(p.PriceCents)/100.0, p.StockQuantity, p.Status)
			skus = append(skus, p.SKU)
		}
		return sb.String(), skus
	default:
		return fmt.Sprintf("unknown tool: %s", name), nil
	}
}

// ExplainProduct generates a short, grounded explanation of a single product.
func (s *AIService) ExplainProduct(ctx context.Context, productID string) (string, error) {
	if !s.llm.Available {
		return "", ErrLLMUnavailable
	}

	p, err := s.product.GetByID(ctx, productID)
	if err != nil {
		return "", fmt.Errorf("failed to look up product: %w", err)
	}

	prompt := fmt.Sprintf(
		"Explain this product to a shopper in 2-3 sentences, based only on the facts given. "+
			"Name: %s. SKU: %s. Description: %s. Price: $%.2f. Stock: %d units. Status: %s. Tags: %s.",
		p.Name, p.SKU, p.Description, float64(p.PriceCents)/100.0, p.StockQuantity, p.Status, strings.Join(p.Tags, ", "),
	)

	resp, err := s.llm.SDK.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(s.llm.Model),
		MaxTokens: 512,
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
	})
	if err != nil {
		return "", fmt.Errorf("assistant call failed: %w", err)
	}

	var sb strings.Builder
	for _, block := range resp.Content {
		if variant, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(variant.Text)
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

type CartNudge struct {
	UserID  string `json:"userId"`
	CartID  string `json:"cartId"`
	Message string `json:"message"`
}

// GenerateAbandonedCartNudges fetches carts abandoned for at least
// sinceMinutes and generates a short re-engagement message for each.
func (s *AIService) GenerateAbandonedCartNudges(ctx context.Context, sinceMinutes int) ([]CartNudge, error) {
	if !s.llm.Available {
		return nil, ErrLLMUnavailable
	}

	carts, err := s.cart.ListAbandoned(ctx, sinceMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to list abandoned carts: %w", err)
	}

	nudges := make([]CartNudge, 0, len(carts))
	for _, c := range carts {
		if len(c.Items) == 0 {
			continue
		}

		var items strings.Builder
		for _, item := range c.Items {
			fmt.Fprintf(&items, "- %dx %s ($%.2f each)\n", item.Quantity, item.ProductName, item.UnitPrice)
		}

		prompt := fmt.Sprintf(
			"Write a short (1-2 sentence), friendly cart-abandonment reminder for a shopper who left "+
				"these items in their cart:\n%s\nDo not invent a discount or promotion.", items.String())

		resp, err := s.llm.SDK.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(s.llm.Model),
			MaxTokens: 256,
			Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
		})
		if err != nil {
			log.Printf("ai-service: nudge generation failed for cart %s: %v", c.ID, err)
			continue
		}

		var sb strings.Builder
		for _, block := range resp.Content {
			if variant, ok := block.AsAny().(anthropic.TextBlock); ok {
				sb.WriteString(variant.Text)
			}
		}
		nudges = append(nudges, CartNudge{UserID: c.UserID, CartID: c.ID, Message: strings.TrimSpace(sb.String())})
	}

	return nudges, nil
}
