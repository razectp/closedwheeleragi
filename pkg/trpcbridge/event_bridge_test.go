package trpcbridge

import (
	"context"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
)

func TestConsumeEvents_Text(t *testing.T) {
	ch := make(chan *event.Event, 10)

	// Partial streaming chunk
	ch <- &event.Event{
		Response: &model.Response{
			IsPartial: true,
			Choices: []model.Choice{
				{Delta: model.Message{Content: "Hello "}},
			},
		},
		Author: "executor",
	}

	// Final response
	ch <- &event.Event{
		Response: &model.Response{
			Done: true,
			Choices: []model.Choice{
				{Message: model.Message{Content: "Hello world"}},
			},
		},
		Author: "executor",
	}
	close(ch)

	var streamedContent string
	var streamDone bool

	eb := NewEventBridge(
		func(content, thinking string, done bool) {
			streamedContent += content
			if done {
				streamDone = true
			}
		},
		nil, nil, nil, nil, nil, nil,
	)

	result, err := eb.ConsumeEvents(context.Background(), ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", result)
	}
	if streamedContent != "Hello " {
		t.Errorf("streamed content mismatch: %q", streamedContent)
	}
	if !streamDone {
		t.Error("stream done was not signaled")
	}
}

func TestConsumeEvents_ToolCalls(t *testing.T) {
	ch := make(chan *event.Event, 10)

	ch <- &event.Event{
		Response: &model.Response{
			Choices: []model.Choice{
				{
					Message: model.Message{
						ToolCalls: []model.ToolCall{
							{
								Function: model.FunctionDefinitionParam{
									Name:      "read_file",
									Arguments: []byte(`{"path":"test.go"}`),
								},
							},
						},
					},
				},
			},
		},
		Author: "researcher",
	}

	// Done response
	ch <- &event.Event{
		Response: &model.Response{
			Done: true,
			Choices: []model.Choice{
				{Message: model.Message{Content: "done"}},
			},
		},
		Author: "researcher",
	}
	close(ch)

	var toolName, toolArgs string
	eb := NewEventBridge(
		nil,
		func(name, args string) {
			toolName = name
			toolArgs = args
		},
		nil, nil, nil, nil, nil,
	)

	_, err := eb.ConsumeEvents(context.Background(), ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if toolName != "read_file" {
		t.Errorf("expected tool name 'read_file', got %q", toolName)
	}
	if toolArgs != `{"path":"test.go"}` {
		t.Errorf("tool args mismatch: %q", toolArgs)
	}
}

func TestConsumeEvents_PipelineRoles(t *testing.T) {
	ch := make(chan *event.Event, 10)

	roles := []string{"planner", "researcher", "executor", "critic"}
	for _, role := range roles {
		ch <- &event.Event{
			Response: &model.Response{
				Done: true,
				Choices: []model.Choice{
					{Message: model.Message{Content: role + " output"}},
				},
			},
			Author: role,
		}
	}
	close(ch)

	var transitions []string
	eb := NewEventBridge(
		nil, nil, nil, nil, nil,
		func(role, status string) {
			transitions = append(transitions, role+":"+status)
		},
		nil,
	)

	_, err := eb.ConsumeEvents(context.Background(), ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each role should produce a "thinking" and "done" transition.
	expected := []string{
		"planner:thinking", "planner:done",
		"researcher:thinking", "researcher:done",
		"executor:thinking", "executor:done",
		"critic:thinking", "critic:done",
	}

	if len(transitions) != len(expected) {
		t.Fatalf("expected %d transitions, got %d: %v", len(expected), len(transitions), transitions)
	}
	for i, exp := range expected {
		if transitions[i] != exp {
			t.Errorf("transition %d: expected %q, got %q", i, exp, transitions[i])
		}
	}
}

func TestConsumeEvents_Error(t *testing.T) {
	ch := make(chan *event.Event, 1)

	ch <- &event.Event{
		Response: &model.Response{
			Error: &model.ResponseError{
				Type:    "api_error",
				Message: "rate limited",
			},
		},
	}
	close(ch)

	eb := NewEventBridge(nil, nil, nil, nil, nil, nil, nil)

	_, err := eb.ConsumeEvents(context.Background(), ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "agent error (api_error): rate limited" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConsumeEvents_Cancel(t *testing.T) {
	ch := make(chan *event.Event) // unbuffered — will block

	ctx, cancel := context.WithCancel(context.Background())

	eb := NewEventBridge(nil, nil, nil, nil, nil, nil, nil)

	done := make(chan struct{})
	var result string
	var err error

	go func() {
		result, err = eb.ConsumeEvents(ctx, ch)
		close(done)
	}()

	// Cancel after a short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Good — it returned
	case <-time.After(2 * time.Second):
		t.Fatal("ConsumeEvents did not return after context cancellation")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestConsumeEvents_ChannelClosed(t *testing.T) {
	ch := make(chan *event.Event)
	close(ch)

	eb := NewEventBridge(nil, nil, nil, nil, nil, nil, nil)
	result, err := eb.ConsumeEvents(context.Background(), ch)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result from closed channel, got %q", result)
	}
}
