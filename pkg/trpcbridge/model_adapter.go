// Package trpcbridge adapts ClosedWheeler's LLM, tool, and session components
// to trpc-agent-go interfaces, enabling multi-agent orchestration via ChainAgent.
package trpcbridge

import (
	"context"
	"sync"

	"ClosedWheeler/pkg/llm"
	"ClosedWheeler/pkg/logger"

	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/tool"
)

// ModelAdapter wraps an llm.Client to implement model.Model.
type ModelAdapter struct {
	client *llm.Client
	name   string
	logger *logger.Logger
	mu     sync.Mutex // protects reasoning effort swap
}

// NewModelAdapter creates a ModelAdapter from an existing llm.Client.
func NewModelAdapter(client *llm.Client, name string, log *logger.Logger) *ModelAdapter {
	return &ModelAdapter{
		client: client,
		name:   name,
		logger: log,
	}
}

// Info returns model metadata.
func (m *ModelAdapter) Info() model.Info {
	return model.Info{Name: m.name}
}

// GenerateContent converts a trpc-agent-go Request into an llm.Client call and
// streams Response objects back on a channel.
func (m *ModelAdapter) GenerateContent(ctx context.Context, req *model.Request) (<-chan *model.Response, error) {
	messages := trpcMessagesToLLM(req.Messages)
	toolDefs := trpcToolsToLLMDefs(req.Tools)

	temp := req.GenerationConfig.Temperature
	topP := req.GenerationConfig.TopP
	maxTok := req.GenerationConfig.MaxTokens

	// Temporarily swap reasoning effort if the request specifies one.
	if req.GenerationConfig.ReasoningEffort != nil {
		m.mu.Lock()
		prev := m.client.GetReasoningEffort()
		m.client.SetReasoningEffort(*req.GenerationConfig.ReasoningEffort)
		defer func() {
			m.client.SetReasoningEffort(prev)
			m.mu.Unlock()
		}()
	}

	ch := make(chan *model.Response, 64)

	if req.GenerationConfig.Stream {
		go m.generateStreaming(ctx, messages, toolDefs, temp, topP, maxTok, ch)
	} else {
		go m.generateBlocking(ctx, messages, toolDefs, temp, topP, maxTok, ch)
	}

	return ch, nil
}

// generateBlocking performs a single blocking LLM call and sends one final Response.
func (m *ModelAdapter) generateBlocking(
	ctx context.Context,
	messages []llm.Message,
	toolDefs []llm.ToolDefinition,
	temp, topP *float64,
	maxTok *int,
	ch chan<- *model.Response,
) {
	defer close(ch)

	resp, err := m.client.ChatWithToolsContext(ctx, messages, toolDefs, temp, topP, maxTok)
	if err != nil {
		sendErrorResponse(ch, err)
		return
	}

	trpcResp := llmResponseToTrpc(resp)
	trpcResp.Done = true
	select {
	case ch <- trpcResp:
	case <-ctx.Done():
	}
}

// generateStreaming performs a streaming LLM call, emitting partial Responses.
func (m *ModelAdapter) generateStreaming(
	ctx context.Context,
	messages []llm.Message,
	toolDefs []llm.ToolDefinition,
	temp, topP *float64,
	maxTok *int,
	ch chan<- *model.Response,
) {
	defer close(ch)

	callback := func(content, thinking string, done bool) {
		partial := &model.Response{
			IsPartial: !done,
			Done:      done,
			Object:    model.ObjectTypeChatCompletionChunk,
			Choices: []model.Choice{
				{
					Delta: model.Message{
						Role:             model.RoleAssistant,
						Content:          content,
						ReasoningContent: thinking,
					},
				},
			},
		}
		select {
		case ch <- partial:
		case <-ctx.Done():
		}
	}

	resp, err := m.client.ChatWithStreamingContext(ctx, messages, toolDefs, temp, topP, maxTok, callback)
	if err != nil {
		sendErrorResponse(ch, err)
		return
	}

	// Send a final complete response with full content and usage.
	final := llmResponseToTrpc(resp)
	final.Done = true
	final.Object = model.ObjectTypeChatCompletion
	select {
	case ch <- final:
	case <-ctx.Done():
	}
}

// sendErrorResponse sends an error Response on the channel.
func sendErrorResponse(ch chan<- *model.Response, err error) {
	errMsg := err.Error()
	ch <- &model.Response{
		Done: true,
		Error: &model.ResponseError{
			Message: errMsg,
			Type:    model.ErrorTypeAPIError,
		},
	}
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

// trpcMessagesToLLM converts trpc-agent-go Messages to llm.Message slice.
func trpcMessagesToLLM(msgs []model.Message) []llm.Message {
	out := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		msg := llm.Message{
			Role:    string(m.Role),
			Content: m.Content,
		}
		if m.ToolID != "" {
			msg.ToolCallID = m.ToolID
		}
		if m.ReasoningContent != "" {
			msg.Thinking = m.ReasoningContent
		}
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]llm.ToolCall, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				msg.ToolCalls[i] = llm.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: llm.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: string(tc.Function.Arguments),
					},
				}
			}
		}
		out = append(out, msg)
	}
	return out
}

// llmResponseToTrpc converts an llm.ChatResponse to a trpc-agent-go Response.
func llmResponseToTrpc(resp *llm.ChatResponse) *model.Response {
	if resp == nil {
		return &model.Response{Done: true}
	}

	trpcResp := &model.Response{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
	}

	if resp.Usage.TotalTokens > 0 {
		trpcResp.Usage = &model.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	trpcResp.Choices = make([]model.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		trpcChoice := model.Choice{
			Index: c.Index,
			Message: model.Message{
				Role:             model.Role(c.Message.Role),
				Content:          c.Message.Content,
				ReasoningContent: c.Message.Thinking,
			},
		}
		if c.FinishReason != "" {
			fr := c.FinishReason
			trpcChoice.FinishReason = &fr
		}
		if len(c.Message.ToolCalls) > 0 {
			trpcChoice.Message.ToolCalls = llmToolCallsToTrpc(c.Message.ToolCalls)
		}
		trpcResp.Choices[i] = trpcChoice
	}

	return trpcResp
}

// llmToolCallsToTrpc converts llm.ToolCall slice to model.ToolCall slice.
func llmToolCallsToTrpc(tcs []llm.ToolCall) []model.ToolCall {
	out := make([]model.ToolCall, len(tcs))
	for i, tc := range tcs {
		out[i] = model.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: model.FunctionDefinitionParam{
				Name:      tc.Function.Name,
				Arguments: []byte(tc.Function.Arguments),
			},
		}
	}
	return out
}

// trpcToolsToLLMDefs converts the trpc-agent-go tool map to llm.ToolDefinition slice.
func trpcToolsToLLMDefs(trpcTools map[string]tool.Tool) []llm.ToolDefinition {
	if len(trpcTools) == 0 {
		return nil
	}

	defs := make([]llm.ToolDefinition, 0, len(trpcTools))
	for _, t := range trpcTools {
		decl := t.Declaration()
		if decl == nil {
			continue
		}

		// Convert tool.Schema to a generic map for FunctionSchema.Parameters.
		var params any
		if decl.InputSchema != nil {
			params = schemaToMap(decl.InputSchema)
		}

		defs = append(defs, llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionSchema{
				Name:        decl.Name,
				Description: decl.Description,
				Parameters:  params,
			},
		})
	}
	return defs
}

// schemaToMap converts a tool.Schema to a JSON-friendly map representation.
func schemaToMap(s *tool.Schema) map[string]any {
	if s == nil {
		return nil
	}

	m := map[string]any{}
	if s.Type != "" {
		m["type"] = s.Type
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}
	if len(s.Properties) > 0 {
		props := map[string]any{}
		for k, v := range s.Properties {
			props[k] = schemaToMap(v)
		}
		m["properties"] = props
	}
	if s.Items != nil {
		m["items"] = schemaToMap(s.Items)
	}
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}
	if s.Default != nil {
		m["default"] = s.Default
	}
	return m
}

