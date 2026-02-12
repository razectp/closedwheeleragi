// Package mcp provides MCP (Model Context Protocol) client integration.
// It connects to external MCP servers, discovers their tools, and bridges
// them into the agent's tool registry so the LLM can call them seamlessly.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"ClosedWheeler/pkg/tools"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ServerConfig describes a single MCP server connection.
type ServerConfig struct {
	// Name is a unique human-readable label for this server.
	Name string `json:"name"`
	// Transport is "stdio" or "sse".
	Transport string `json:"transport"`
	// Command is the executable path (stdio transport only).
	Command string `json:"command,omitempty"`
	// Args are command-line arguments (stdio transport only).
	Args []string `json:"args,omitempty"`
	// Env are environment variables for the subprocess (stdio transport only).
	Env []string `json:"env,omitempty"`
	// URL is the SSE endpoint (sse transport only).
	URL string `json:"url,omitempty"`
	// Enabled controls whether this server is active.
	Enabled bool `json:"enabled"`
}

// ServerInfo holds runtime information about a connected MCP server.
type ServerInfo struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Connected bool     `json:"connected"`
	Tools     []string `json:"tools"`
	Error     string   `json:"error,omitempty"`
}

// Manager handles MCP server connections and tool bridging.
type Manager struct {
	registry *tools.Registry
	mu       sync.RWMutex
	clients  map[string]*mcpclient.Client // name -> client
	servers  []ServerConfig               // configured servers
	infos    map[string]*ServerInfo        // name -> runtime info
	toolMap  map[string]string             // tool_name -> server_name (for cleanup)
}

// NewManager creates a new MCP manager.
func NewManager(registry *tools.Registry) *Manager {
	return &Manager{
		registry: registry,
		clients:  make(map[string]*mcpclient.Client),
		infos:    make(map[string]*ServerInfo),
		toolMap:  make(map[string]string),
	}
}

// Configure sets the server list. Does not connect; call ConnectAll() after.
func (m *Manager) Configure(servers []ServerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers = servers
}

// ConnectAll connects to all enabled servers and registers their tools.
func (m *Manager) ConnectAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.servers {
		if !m.servers[i].Enabled {
			continue
		}
		m.connectServer(&m.servers[i])
	}
}

// DisconnectAll closes all MCP server connections and unregisters their tools.
func (m *Manager) DisconnectAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, c := range m.clients {
		_ = c.Close()
		delete(m.clients, name)
	}

	// Unregister all MCP-sourced tools
	for toolName := range m.toolMap {
		m.registry.Unregister(toolName)
	}
	m.toolMap = make(map[string]string)
	m.infos = make(map[string]*ServerInfo)
}

// Reload disconnects everything, then reconnects all enabled servers.
func (m *Manager) Reload() {
	m.DisconnectAll()
	m.ConnectAll()
}

// ListServers returns runtime information about all servers.
func (m *Manager) ListServers() []ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]ServerInfo, 0, len(m.servers))
	for _, s := range m.servers {
		if info, ok := m.infos[s.Name]; ok {
			out = append(out, *info)
		} else {
			out = append(out, ServerInfo{
				Name:      s.Name,
				Transport: s.Transport,
				Connected: false,
			})
		}
	}
	return out
}

// ServerCount returns the number of configured servers.
func (m *Manager) ServerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.servers)
}

// ToolCount returns the total number of MCP-sourced tools currently registered.
func (m *Manager) ToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.toolMap)
}

// GetConfigs returns the current server configurations.
func (m *Manager) GetConfigs() []ServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ServerConfig, len(m.servers))
	copy(out, m.servers)
	return out
}

// AddServer adds a server config and optionally connects immediately.
func (m *Manager) AddServer(cfg ServerConfig, connectNow bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check duplicate name
	for _, s := range m.servers {
		if s.Name == cfg.Name {
			return fmt.Errorf("server %q already exists", cfg.Name)
		}
	}

	m.servers = append(m.servers, cfg)
	if connectNow && cfg.Enabled {
		m.connectServer(&m.servers[len(m.servers)-1])
	}
	return nil
}

// RemoveServer disconnects and removes a server by name.
func (m *Manager) RemoveServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, s := range m.servers {
		if s.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("server %q not found", name)
	}

	// Disconnect if connected
	if c, ok := m.clients[name]; ok {
		_ = c.Close()
		delete(m.clients, name)
	}

	// Unregister tools from this server
	for toolName, srvName := range m.toolMap {
		if srvName == name {
			m.registry.Unregister(toolName)
			delete(m.toolMap, toolName)
		}
	}

	delete(m.infos, name)
	m.servers = append(m.servers[:idx], m.servers[idx+1:]...)
	return nil
}

// connectServer connects to a single MCP server and registers its tools.
// Must be called with m.mu held.
func (m *Manager) connectServer(cfg *ServerConfig) {
	info := &ServerInfo{
		Name:      cfg.Name,
		Transport: cfg.Transport,
	}
	m.infos[cfg.Name] = info

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var (
		c   *mcpclient.Client
		err error
	)

	switch cfg.Transport {
	case "stdio":
		c, err = mcpclient.NewStdioMCPClient(cfg.Command, cfg.Env, cfg.Args...)
	case "sse":
		c, err = mcpclient.NewSSEMCPClient(cfg.URL)
		if err == nil {
			err = c.Start(ctx)
		}
	default:
		info.Error = fmt.Sprintf("unsupported transport: %s", cfg.Transport)
		log.Printf("[MCP] %s: %s", cfg.Name, info.Error)
		return
	}

	if err != nil {
		info.Error = fmt.Sprintf("connection failed: %v", err)
		log.Printf("[MCP] %s: %s", cfg.Name, info.Error)
		return
	}

	// Initialize the MCP session
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "ClosedWheelerAGI",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initReq)
	if err != nil {
		info.Error = fmt.Sprintf("initialize failed: %v", err)
		log.Printf("[MCP] %s: %s", cfg.Name, info.Error)
		_ = c.Close()
		return
	}

	// List tools from this server
	toolsResult, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		info.Error = fmt.Sprintf("list tools failed: %v", err)
		log.Printf("[MCP] %s: %s", cfg.Name, info.Error)
		_ = c.Close()
		return
	}

	m.clients[cfg.Name] = c
	info.Connected = true

	// Bridge each MCP tool into our tool registry
	for _, mcpTool := range toolsResult.Tools {
		toolName := fmt.Sprintf("mcp_%s_%s", cfg.Name, mcpTool.Name)
		info.Tools = append(info.Tools, toolName)

		desc := mcpTool.Description
		if desc == "" {
			desc = fmt.Sprintf("MCP tool %s from server %s", mcpTool.Name, cfg.Name)
		} else {
			desc = fmt.Sprintf("[MCP:%s] %s", cfg.Name, desc)
		}

		// Convert MCP input schema to our JSONSchema
		params := convertMCPSchema(mcpTool.InputSchema)

		// Capture loop variables for closure
		serverName := cfg.Name
		remoteName := mcpTool.Name

		tool := &tools.Tool{
			Name:        toolName,
			Description: desc,
			Parameters:  params,
			Handler: func(args map[string]any) (tools.ToolResult, error) {
				return m.callMCPTool(serverName, remoteName, args)
			},
		}

		if err := m.registry.Register(tool); err != nil {
			log.Printf("[MCP] %s: failed to register tool %s: %v", cfg.Name, toolName, err)
			continue
		}
		m.toolMap[toolName] = cfg.Name
	}

	log.Printf("[MCP] %s: connected, %d tools registered", cfg.Name, len(info.Tools))
}

// callMCPTool invokes a tool on an MCP server.
func (m *Manager) callMCPTool(serverName, toolName string, args map[string]any) (tools.ToolResult, error) {
	m.mu.RLock()
	c, ok := m.clients[serverName]
	m.mu.RUnlock()

	if !ok {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("MCP server %q not connected", serverName),
		}, fmt.Errorf("MCP server %q not connected", serverName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = args

	result, err := c.CallTool(ctx, req)
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("MCP call failed: %v", err),
		}, err
	}

	// Extract text from result content
	output := extractContent(result)

	if result.IsError {
		return tools.ToolResult{
			Success: false,
			Error:   output,
		}, nil
	}

	return tools.ToolResult{
		Success: true,
		Output:  output,
	}, nil
}

// extractContent extracts text from MCP CallToolResult content blocks.
func extractContent(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	var parts []string
	for _, c := range result.Content {
		// Content is an interface; try JSON marshal/unmarshal to extract text
		raw, err := json.Marshal(c)
		if err != nil {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		if text, ok := m["text"].(string); ok {
			parts = append(parts, text)
		}
	}

	if len(parts) == 0 {
		// Fallback: marshal the whole result
		raw, _ := json.Marshal(result.Content)
		return string(raw)
	}

	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "\n"
		}
		out += p
	}
	return out
}

// convertMCPSchema converts an MCP tool's InputSchema to our tools.JSONSchema.
func convertMCPSchema(schema mcp.ToolInputSchema) *tools.JSONSchema {
	js := &tools.JSONSchema{
		Type:     schema.Type,
		Required: schema.Required,
	}

	if len(schema.Properties) > 0 {
		js.Properties = make(map[string]tools.Property, len(schema.Properties))
		for name, raw := range schema.Properties {
			prop := tools.Property{}
			// raw may be a map or json.RawMessage depending on mcp-go version
			switch v := raw.(type) {
			case map[string]any:
				if t, ok := v["type"].(string); ok {
					prop.Type = t
				}
				if d, ok := v["description"].(string); ok {
					prop.Description = d
				}
			case []byte:
				var pm map[string]any
				if err := json.Unmarshal(v, &pm); err == nil {
					if t, ok := pm["type"].(string); ok {
						prop.Type = t
					}
					if d, ok := pm["description"].(string); ok {
						prop.Description = d
					}
				}
			}
			js.Properties[name] = prop
		}
	}

	return js
}
