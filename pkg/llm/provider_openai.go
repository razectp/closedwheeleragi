package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ClosedWheeler/pkg/config"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs.
type OpenAIProvider struct {
	oauth           *config.OAuthCredentials
	reasoningEffort string // "low", "medium", "high", "xhigh"
}

func (p *OpenAIProvider) Name() string { return "openai" }

// SetOAuth sets OAuth credentials for the OpenAI provider.
func (p *OpenAIProvider) SetOAuth(creds *config.OAuthCredentials) { p.oauth = creds }

// GetOAuth returns the current OAuth credentials.
func (p *OpenAIProvider) GetOAuth() *config.OAuthCredentials { return p.oauth }

// RefreshIfNeeded refreshes the OAuth token if it's close to expiry.
func (p *OpenAIProvider) RefreshIfNeeded() {
	if p.oauth == nil || !p.oauth.NeedsRefresh() || p.oauth.RefreshToken == "" {
		return
	}
	var newCreds *config.OAuthCredentials
	var err error
	switch p.oauth.Provider {
	case "google":
		newCreds, err = RefreshGoogleToken(p.oauth.RefreshToken)
		if err == nil && newCreds != nil {
			newCreds.ProjectID = p.oauth.ProjectID // preserve projectID
		}
	default:
		newCreds, err = RefreshOpenAIToken(p.oauth.RefreshToken)
	}
	if err != nil || newCreds == nil {
		return
	}
	p.oauth = newCreds
	_ = config.SaveOAuth(newCreds)
}

func (p *OpenAIProvider) Endpoint(baseURL string) string {
	return baseURL + "/chat/completions"
}

func (p *OpenAIProvider) SetHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Content-Type", "application/json")
	// OAuth Bearer token takes priority over API key
	if p.oauth != nil && p.oauth.AccessToken != "" && !p.oauth.IsExpired() {
		req.Header.Set("Authorization", "Bearer "+p.oauth.AccessToken)
		switch p.oauth.Provider {
		case "openai":
			if p.oauth.AccountID != "" {
				req.Header.Set("ChatGPT-Account-Id", p.oauth.AccountID)
			}
			req.Header.Set("originator", "codex_cli_rs")
		case "google":
			if p.oauth.ProjectID != "" {
				req.Header.Set("x-goog-user-project", p.oauth.ProjectID)
			}
		}
	} else {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}

func (p *OpenAIProvider) BuildRequestBody(model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, stream bool) ([]byte, error) {
	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		Tools:       tools,
		Temperature: temperature,
		TopP:        topP,
		MaxTokens:   maxTokens,
		Stream:      stream,
	}

	if p.reasoningEffort != "" {
		reqBody.ReasoningEffort = p.reasoningEffort
	}

	if len(tools) > 0 {
		reqBody.ToolChoice = "auto"
	}

	return json.Marshal(reqBody)
}

// SetReasoningEffort sets the reasoning effort level for reasoning models.
func (p *OpenAIProvider) SetReasoningEffort(effort string) { p.reasoningEffort = effort }

// GetReasoningEffort returns the current reasoning effort level.
func (p *OpenAIProvider) GetReasoningEffort() string { return p.reasoningEffort }

func (p *OpenAIProvider) ParseResponseBody(body []byte) (*ChatResponse, error) {
	var resp ChatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &resp, nil
}

func (p *OpenAIProvider) ParseRateLimits(h http.Header) RateLimits {
	rl := RateLimits{}
	if v := h.Get("x-ratelimit-remaining-requests"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil {
			log.Printf("[WARN] Failed to parse x-ratelimit-remaining-requests: %v (value: %s)", err, v)
		} else {
			rl.RemainingRequests = val
		}
	}
	if v := h.Get("x-ratelimit-remaining-tokens"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil {
			log.Printf("[WARN] Failed to parse x-ratelimit-remaining-tokens: %v (value: %s)", err, v)
		} else {
			rl.RemainingTokens = val
		}
	}
	if v := h.Get("x-ratelimit-reset-requests"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			rl.ResetRequests = time.Now().Add(d)
		}
	}
	if v := h.Get("x-ratelimit-reset-tokens"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			rl.ResetTokens = time.Now().Add(d)
		}
	}
	return rl
}

func (p *OpenAIProvider) ParseSSEStream(body io.Reader, callback StreamingCallback) (*ChatResponse, error) {
	reader := bufio.NewReader(body)

	var fullContent strings.Builder
	var toolCalls []ToolCall
	var lastResponse StreamingResponse
	var finishReason string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			if callback != nil {
				callback("", true)
			}
			break
		}

		var streamResp StreamingResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			log.Printf("[WARN] Skipping malformed streaming chunk: %v (data: %s)", err, data)
			continue
		}

		lastResponse = streamResp

		if len(streamResp.Choices) > 0 {
			choice := streamResp.Choices[0]

			// Capture finish_reason from the stream
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}

			if choice.Delta.Content != "" {
				fullContent.WriteString(choice.Delta.Content)
				if callback != nil {
					callback(choice.Delta.Content, false)
				}
			}

			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					if tc.ID != "" {
						toolCalls = append(toolCalls, tc)
					} else if len(toolCalls) > 0 {
						last := &toolCalls[len(toolCalls)-1]
						last.Function.Arguments += tc.Function.Arguments
					}
				}
			}
		}
	}

	if finishReason == "" {
		finishReason = "stop"
	}

	finalResponse := &ChatResponse{
		ID:      lastResponse.ID,
		Object:  "chat.completion",
		Created: lastResponse.Created,
		Model:   lastResponse.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:      "assistant",
					Content:   fullContent.String(),
					ToolCalls: toolCalls,
				},
				FinishReason: finishReason,
			},
		},
	}

	return finalResponse, nil
}

func (p *OpenAIProvider) SupportsModelListing() bool { return true }
