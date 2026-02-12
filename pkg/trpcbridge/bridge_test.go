package trpcbridge

import (
	"context"
	"testing"

	"ClosedWheeler/pkg/tools"

	"trpc.group/trpc-go/trpc-agent-go/session"
)

func TestNewBridge(t *testing.T) {
	registry := tools.NewRegistry()
	executor := tools.NewExecutor(registry)

	bridge := NewBridge(nil, registry, executor, nil)
	if bridge == nil {
		t.Fatal("bridge is nil")
	}
	if bridge.ModelAdapter() == nil {
		t.Error("model adapter is nil")
	}
	if bridge.SessionAdapter() == nil {
		t.Error("session adapter is nil")
	}
}

func TestNewBridge_WithTools(t *testing.T) {
	registry := tools.NewRegistry()

	toolNames := []string{"read_file", "write_file", "list_files"}
	for _, name := range toolNames {
		n := name
		err := registry.Register(&tools.Tool{
			Name:        n,
			Description: "test " + n,
			Parameters:  &tools.JSONSchema{Type: "object"},
			Handler: func(args map[string]any) (tools.ToolResult, error) {
				return tools.ToolResult{Success: true}, nil
			},
		})
		if err != nil {
			t.Fatalf("register %s: %v", n, err)
		}
	}
	executor := tools.NewExecutor(registry)

	bridge := NewBridge(nil, registry, executor, nil)

	// allTools should have 3
	if len(bridge.allTools) != 3 {
		t.Errorf("expected 3 allTools, got %d", len(bridge.allTools))
	}

	// readOnlyTools should have 2 (read_file, list_files)
	if len(bridge.readOnlyTools) != 2 {
		t.Errorf("expected 2 readOnlyTools, got %d", len(bridge.readOnlyTools))
	}
}

func TestNewChainPipeline(t *testing.T) {
	registry := tools.NewRegistry()
	executor := tools.NewExecutor(registry)

	bridge := NewBridge(nil, registry, executor, nil)

	rolePrompt := func(role string) string {
		return "You are the " + role
	}

	chain := bridge.NewChainPipeline(rolePrompt)
	if chain == nil {
		t.Fatal("chain is nil")
	}

	info := chain.Info()
	if info.Name != "pipeline" {
		t.Errorf("expected chain name 'pipeline', got %s", info.Name)
	}

	subAgents := chain.SubAgents()
	if len(subAgents) != 4 {
		t.Fatalf("expected 4 sub-agents, got %d", len(subAgents))
	}

	expectedNames := []string{"planner", "researcher", "executor", "critic"}
	for i, ag := range subAgents {
		agInfo := ag.Info()
		if agInfo.Name != expectedNames[i] {
			t.Errorf("sub-agent %d: expected %q, got %q", i, expectedNames[i], agInfo.Name)
		}
	}
}

func TestNewLLMAgent(t *testing.T) {
	registry := tools.NewRegistry()
	executor := tools.NewExecutor(registry)

	bridge := NewBridge(nil, registry, executor, nil)
	agent := bridge.NewLLMAgent("test-agent", "Be helpful", nil)

	if agent == nil {
		t.Fatal("agent is nil")
	}
	info := agent.Info()
	if info.Name != "test-agent" {
		t.Errorf("expected 'test-agent', got %s", info.Name)
	}
}

func TestNewLLMAgent_WithTools(t *testing.T) {
	registry := tools.NewRegistry()
	err := registry.Register(&tools.Tool{
		Name:        "my_tool",
		Description: "test tool",
		Parameters:  &tools.JSONSchema{Type: "object"},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			return tools.ToolResult{Success: true}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	executor := tools.NewExecutor(registry)

	bridge := NewBridge(nil, registry, executor, nil)
	agent := bridge.NewLLMAgent("tool-agent", "Use tools", bridge.allTools)

	agTools := agent.Tools()
	if len(agTools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(agTools))
	}
}

func TestNewRunner(t *testing.T) {
	registry := tools.NewRegistry()
	executor := tools.NewExecutor(registry)
	bridge := NewBridge(nil, registry, executor, nil)

	rolePrompt := func(role string) string { return role }
	chain := bridge.NewChainPipeline(rolePrompt)

	r := bridge.NewRunner("test-app", chain)
	if r == nil {
		t.Fatal("runner is nil")
	}
}

func TestBridgeClose(t *testing.T) {
	registry := tools.NewRegistry()
	executor := tools.NewExecutor(registry)
	bridge := NewBridge(nil, registry, executor, nil)

	if err := bridge.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestSessionAdapterCreateAndGet(t *testing.T) {
	sa := NewSessionAdapter()
	defer sa.Close()

	key := session.Key{
		AppName:   "test",
		UserID:    "user1",
		SessionID: "sess1",
	}

	ctx := context.Background()

	sess, err := sa.CreateSession(ctx, key, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if sess == nil {
		t.Fatal("session is nil")
	}

	// Get should return the same session.
	got, err := sa.GetSession(ctx, key)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != sess {
		t.Error("expected same session pointer")
	}
}

func TestSessionAdapterCreateDuplicate(t *testing.T) {
	sa := NewSessionAdapter()
	defer sa.Close()

	key := session.Key{
		AppName:   "test",
		UserID:    "user1",
		SessionID: "sess1",
	}

	ctx := context.Background()

	sess1, _ := sa.CreateSession(ctx, key, nil)
	sess2, _ := sa.CreateSession(ctx, key, nil)

	// Should return the same session (not create a new one).
	if sess1 != sess2 {
		t.Error("duplicate create should return existing session")
	}
}

func TestSessionAdapterGetNotFound(t *testing.T) {
	sa := NewSessionAdapter()
	defer sa.Close()

	key := session.Key{
		AppName:   "test",
		UserID:    "user1",
		SessionID: "nonexistent",
	}

	sess, err := sa.GetSession(context.Background(), key)
	if err != nil {
		t.Errorf("expected nil error for nonexistent session, got %v", err)
	}
	if sess != nil {
		t.Error("expected nil session for nonexistent key")
	}
}

func TestSessionAdapterDelete(t *testing.T) {
	sa := NewSessionAdapter()
	defer sa.Close()

	key := session.Key{
		AppName:   "test",
		UserID:    "user1",
		SessionID: "sess1",
	}

	ctx := context.Background()
	sa.CreateSession(ctx, key, nil)

	if err := sa.DeleteSession(ctx, key); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	sess, err := sa.GetSession(ctx, key)
	if err != nil {
		t.Errorf("expected nil error after deletion, got %v", err)
	}
	if sess != nil {
		t.Error("expected nil session after deletion")
	}
}

func TestSessionAdapterStubs(t *testing.T) {
	sa := NewSessionAdapter()
	defer sa.Close()

	ctx := context.Background()
	userKey := session.UserKey{AppName: "test", UserID: "user1"}

	// All stubs should succeed without error.
	if err := sa.UpdateAppState(ctx, "test", nil); err != nil {
		t.Error(err)
	}
	if err := sa.DeleteAppState(ctx, "test", "key"); err != nil {
		t.Error(err)
	}
	if _, err := sa.ListAppStates(ctx, "test"); err != nil {
		t.Error(err)
	}
	if err := sa.UpdateUserState(ctx, userKey, nil); err != nil {
		t.Error(err)
	}
	if _, err := sa.ListUserStates(ctx, userKey); err != nil {
		t.Error(err)
	}
	if err := sa.DeleteUserState(ctx, userKey, "key"); err != nil {
		t.Error(err)
	}
	if sessions, err := sa.ListSessions(ctx, userKey); err != nil || sessions != nil {
		t.Error("ListSessions should return nil, nil")
	}
	key := session.Key{AppName: "test", UserID: "user1", SessionID: "s1"}
	if err := sa.UpdateSessionState(ctx, key, nil); err != nil {
		t.Error(err)
	}
	if err := sa.CreateSessionSummary(ctx, nil, "", false); err != nil {
		t.Error(err)
	}
	if err := sa.EnqueueSummaryJob(ctx, nil, "", false); err != nil {
		t.Error(err)
	}
	if text, ok := sa.GetSessionSummaryText(ctx, nil); ok || text != "" {
		t.Error("expected empty summary")
	}
}
