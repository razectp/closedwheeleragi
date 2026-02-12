package trpcbridge

import (
	"context"
	"fmt"
	"strings"

	"ClosedWheeler/pkg/llm"
	"ClosedWheeler/pkg/logger"

	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
)

// PipelineRoleCallback is called when a pipeline role changes status.
// The role parameter is the agent name (e.g. "planner", "researcher").
// The status parameter is "thinking", "done", or "error".
type PipelineRoleCallback func(role string, status string)

// EventBridge translates trpc-agent-go event channels into TUI callbacks.
type EventBridge struct {
	streamCb    llm.StreamingCallback
	toolStartCb func(name, args string)
	toolDoneCb  func(name, result string)
	toolErrCb   func(name string, err error)
	statusCb    func(string)
	pipelineCb  PipelineRoleCallback
	logger      *logger.Logger
}

// NewEventBridge creates an EventBridge with the given callbacks.
// Any callback may be nil; nil callbacks are silently skipped.
func NewEventBridge(
	streamCb llm.StreamingCallback,
	toolStartCb func(string, string),
	toolDoneCb func(string, string),
	toolErrCb func(string, error),
	statusCb func(string),
	pipelineCb PipelineRoleCallback,
	log *logger.Logger,
) *EventBridge {
	return &EventBridge{
		streamCb:    streamCb,
		toolStartCb: toolStartCb,
		toolDoneCb:  toolDoneCb,
		toolErrCb:   toolErrCb,
		statusCb:    statusCb,
		pipelineCb:  pipelineCb,
		logger:      log,
	}
}

// ConsumeEvents reads from the event channel, fires TUI callbacks, and returns
// the accumulated final text from the last agent in the chain.
func (eb *EventBridge) ConsumeEvents(ctx context.Context, events <-chan *event.Event) (string, error) {
	var lastAuthor string
	var finalText strings.Builder

	for {
		select {
		case <-ctx.Done():
			return finalText.String(), ctx.Err()

		case evt, ok := <-events:
			if !ok {
				// Channel closed — return accumulated text.
				return finalText.String(), nil
			}

			// Check for error events.
			if evt.Response != nil && evt.Response.Error != nil {
				return finalText.String(), fmt.Errorf("agent error (%s): %s",
					evt.Response.Error.Type, evt.Response.Error.Message)
			}

			// Track author changes → pipeline role transitions.
			if evt.Author != "" && evt.Author != lastAuthor {
				lastAuthor = evt.Author
				if eb.pipelineCb != nil {
					eb.pipelineCb(lastAuthor, "thinking")
				}
				if eb.statusCb != nil {
					eb.statusCb(fmt.Sprintf("Pipeline: %s thinking...", evt.Author))
				}
			}

			// Process response content.
			if evt.Response == nil {
				continue
			}

			// Handle partial streaming chunks.
			if evt.Response.IsPartial {
				for _, c := range evt.Response.Choices {
					content := c.Delta.Content
					thinking := c.Delta.ReasoningContent
					if (content != "" || thinking != "") && eb.streamCb != nil {
						eb.streamCb(content, thinking, false)
					}
				}
				continue
			}

			// Handle tool calls in responses.
			for _, c := range evt.Response.Choices {
				for _, tc := range c.Message.ToolCalls {
					if eb.toolStartCb != nil {
						eb.toolStartCb(tc.Function.Name, string(tc.Function.Arguments))
					}
				}
			}

			// Handle final (non-partial) response.
			if evt.Response.Done {
				// Collect final text from the last response.
				for _, c := range evt.Response.Choices {
					if c.Message.Content != "" {
						finalText.WriteString(c.Message.Content)
					}
				}

				// Signal stream done.
				if eb.streamCb != nil {
					eb.streamCb("", "", true)
				}

				// Signal role done.
				if lastAuthor != "" && eb.pipelineCb != nil {
					eb.pipelineCb(lastAuthor, "done")
				}
			}

			// Runner completion event — pipeline is finished.
			if evt.IsRunnerCompletion() {
				return finalText.String(), nil
			}

			// Check for Object-based completion signals.
			if evt.Response.Object == model.ObjectTypeRunnerCompletion {
				return finalText.String(), nil
			}
		}
	}
}
