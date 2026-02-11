package llm

import (
	"io"
	"net/http"
	"strings"
)

// Provider abstracts the differences between LLM API protocols.
// The canonical types (ChatResponse, Message, ToolCall, Usage) remain unchanged;
// each provider translates to/from them.
type Provider interface {
	// Name returns the provider identifier (e.g. "openai", "anthropic").
	Name() string

	// Endpoint returns the full URL for chat completion requests.
	Endpoint(baseURL string) string

	// SetHeaders sets provider-specific HTTP headers on the request.
	SetHeaders(req *http.Request, apiKey string)

	// BuildRequestBody converts canonical parameters into the provider's
	// JSON request format. Returns the marshalled JSON bytes.
	BuildRequestBody(model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, stream bool) ([]byte, error)

	// ParseResponseBody parses the provider's JSON response into a
	// canonical ChatResponse.
	ParseResponseBody(body []byte) (*ChatResponse, error)

	// ParseRateLimits extracts rate limit information from response headers.
	ParseRateLimits(h http.Header) RateLimits

	// ParseSSEStream parses a Server-Sent Events stream and calls callback
	// for each content chunk. Returns the assembled ChatResponse.
	ParseSSEStream(body io.Reader, callback StreamingCallback) (*ChatResponse, error)

	// SupportsModelListing returns true if the provider supports the /models endpoint.
	SupportsModelListing() bool
}

// DetectProvider determines the correct provider based on explicit name,
// model name prefix, or API key prefix. Empty providerName triggers auto-detection.
func DetectProvider(providerName, modelName, apiKey string) Provider {
	name := strings.ToLower(strings.TrimSpace(providerName))

	// Explicit provider name
	switch name {
	case "anthropic":
		return &AnthropicProvider{}
	case "openai":
		return &OpenAIProvider{}
	}

	// Auto-detect by model name
	lowerModel := strings.ToLower(modelName)
	if strings.HasPrefix(lowerModel, "claude") {
		return &AnthropicProvider{}
	}

	// Auto-detect by API key prefix
	if strings.HasPrefix(apiKey, "sk-ant-") {
		return &AnthropicProvider{}
	}

	// Default to OpenAI-compatible
	return &OpenAIProvider{}
}
