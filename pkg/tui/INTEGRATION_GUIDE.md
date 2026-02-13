# TUI Integration Guide - ClosedWheelerAGI

## Table of Contents
1. [Quick Start](#quick-start)
2. [Architecture Overview](#architecture-overview)
3. [Integration Patterns](#integration-patterns)
4. [Custom Extensions](#custom-extensions)
5. [Best Practices](#best-practices)
6. [Troubleshooting](#troubleshooting)
7. [Advanced Examples](#advanced-examples)

---

## Quick Start

### Basic TUI Setup

```go
package main

import (
    "log"
    tea "github.com/charmbracelet/bubbletea"
    "ClosedWheeler/pkg/agent"
    "ClosedWheeler/pkg/tui"
)

func main() {
    // 1. Load configuration
    config, err := agent.LoadConfig("config.json")
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // 2. Initialize agent
    ag, err := agent.New(config)
    if err != nil {
        log.Fatal("Failed to create agent:", err)
    }
    
    // 3. Create TUI model
    model := tui.NewEnhancedModel(ag)
    
    // 4. Run the program
    program := tea.NewProgram(model, tea.WithAltScreen())
    if _, err := program.Run(); err != nil {
        log.Fatal("Failed to run TUI:", err)
    }
}
```

### Essential TUI Components

```go
// Core components you'll interact with:
type EnhancedModel struct {
    agent           *agent.Agent           // AI agent instance
    messageQueue    *MessageQueue          // Conversation history
    viewport        viewport.Model         // Main display area
    textarea        textarea.Model         // User input
    spinner         spinner.Model          // Loading animations
    // ... many more components
}
```

---

## Architecture Overview

### Component Hierarchy

```
EnhancedModel (Main TUI)
├── MessageQueue (Thread-safe conversation management)
│   ├── Add()           - Add messages
│   ├── UpdateLast()    - Update streaming messages
│   ├── GetAll()        - Retrieve all messages
│   └── Clear()         - Clear conversation
├── Command System (45+ slash commands)
│   ├── GetAllCommands() - Get command registry
│   ├── FindCommand()   - Lookup commands
│   └── cmd<Name>()     - Command handlers
├── Rendering System
│   ├── renderHeader()      - App title and version
│   ├── renderStatusBar()   - Status indicators
│   ├── renderActiveTools() - Tool execution status
│   └── renderProcessingArea() - Current operation
├── Overlay System
│   ├── Help Menu (F1)
│   ├── Settings (Alt+S)
│   ├── Model Picker (Alt+M)
│   └── Debate Wizard (/debate)
└── Layout Management
    ├── calculateViewportHeight()
    ├── recalculateLayout()
    └── updateViewport()
```

### Event Flow

```
User Input → Update() → Message Processing → View() → Display
     ↓              ↓              ↓           ↓
Key Events → Command Handler → State Update → Render
     ↓              ↓              ↓           ↓
Ctrl+C → Stop Request → Add Message → Show Status
     ↓              ↓              ↓           ↓
Enter → Send Message → Queue Response → Stream Output
```

---

## Integration Patterns

### 1. Adding Custom Commands

```go
// Step 1: Define command in GetAllCommands()
{
    Name:        "custom",
    Aliases:     []string{"cst"},
    Category:    "Custom",
    Description: "Custom command example",
    Usage:       "/custom [arg1] [arg2]",
    Handler:     cmdCustom,
}

// Step 2: Implement handler
func cmdCustom(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
    // Validate arguments
    if len(args) < 2 {
        m.messageQueue.Add(QueuedMessage{
            Role:      "error",
            Content:   "Usage: /custom <arg1> <arg2>",
            Timestamp: time.Now(),
            Complete:  true,
        })
        m.updateViewport()
        return m, nil
    }
    
    // Process command logic
    result := fmt.Sprintf("Processed: %s + %s", args[0], args[1])
    
    // Provide feedback
    m.messageQueue.Add(QueuedMessage{
        Role:      "system",
        Content:   result,
        Timestamp: time.Now(),
        Complete:  true,
    })
    
    m.updateViewport()
    return m, nil
}
```

### 2. Custom Message Types

```go
// Define custom message type
type customProgressMsg struct {
    percent int
    message string
}

// Send custom command
func (m *EnhancedModel) startCustomProcess() tea.Cmd {
    return func() tea.Msg {
        // Simulate work
        time.Sleep(1 * time.Second)
        return customProgressMsg{percent: 50, message: "Processing..."}
    }
}

// Handle in Update()
case customProgressMsg:
    m.status = fmt.Sprintf("Progress: %d%% - %s", msg.percent, msg.message)
    return m, nil
```

### 3. Custom Overlays

```go
// Add to EnhancedModel struct
type EnhancedModel struct {
    // ... existing fields
    customOverlayActive bool
    customOverlayCursor int
    customOverlayItems  []string
}

// Add to View() method
if m.customOverlayActive {
    return m.renderCustomOverlay()
}

// Add to Update() method
if m.customOverlayActive {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyEsc:
            m.customOverlayActive = false
            return m, nil
        case tea.KeyUp:
            m.customOverlayCursor--
            if m.customOverlayCursor < 0 {
                m.customOverlayCursor = len(m.customOverlayItems) - 1
            }
            return m, nil
        case tea.KeyDown:
            m.customOverlayCursor = (m.customOverlayCursor + 1) % len(m.customOverlayItems)
            return m, nil
        case tea.KeyEnter:
            // Handle selection
            selected := m.customOverlayItems[m.customOverlayCursor]
            m.customOverlayActive = false
            return m, m.processCustomSelection(selected)
        }
    }
}

// Implement render method
func (m *EnhancedModel) renderCustomOverlay() string {
    var content strings.Builder
    
    content.WriteString("┌─ Custom Overlay ──────────────────────────┐\n")
    
    for i, item := range m.customOverlayItems {
        cursor := " "
        if i == m.customOverlayCursor {
            cursor = ">"
        }
        content.WriteString(fmt.Sprintf("│ %s %s\n", cursor, item))
    }
    
    content.WriteString("│                                          │\n")
    content.WriteString("│ [↑↓ Navigate] [Enter Select] [Esc Close] │\n")
    content.WriteString("└──────────────────────────────────────────┘")
    
    return content.String()
}
```

### 4. Message Queue Integration

```go
// Thread-safe message addition
func (m *EnhancedModel) addSystemMessage(content string) {
    m.messageQueue.Add(QueuedMessage{
        Role:      "system",
        Content:   content,
        Timestamp: time.Now(),
        Complete:  true,
    })
    m.updateViewport()
}

// Streaming message updates
func (m *EnhancedModel) startStreamingResponse() {
    m.messageQueue.Add(QueuedMessage{
        Role:      "assistant",
        Content:   "",
        Streaming: true,
        Timestamp: time.Now(),
        Complete:  false,
    })
    m.updateViewport()
}

func (m *EnhancedModel) updateStreamingChunk(chunk string) {
    m.messageQueue.UpdateLast(func(qm *QueuedMessage) {
        qm.Content += chunk
        qm.StreamChunk += chunk
    })
    m.updateViewport()
}

func (m *EnhancedModel) completeStreamingResponse(stats *MessageStats) {
    m.messageQueue.UpdateLast(func(qm *QueuedMessage) {
        qm.Complete = true
        qm.Streaming = false
        qm.Stats = stats
    })
    m.updateViewport()
}
```

---

## Custom Extensions

### 1. Plugin System

```go
// Plugin interface
type TUIPlugin interface {
    Name() string
    Initialize(m *EnhancedModel) error
    Commands() []Command
    Cleanup() error
}

// Plugin manager
type PluginManager struct {
    plugins []TUIPlugin
    model   *EnhancedModel
}

func (pm *PluginManager) RegisterPlugin(plugin TUIPlugin) error {
    if err := plugin.Initialize(pm.model); err != nil {
        return err
    }
    pm.plugins = append(pm.plugins, plugin)
    
    // Register plugin commands
    for _, cmd := range plugin.Commands() {
        // Add to command system
    }
    
    return nil
}

// Example plugin
type LoggerPlugin struct{}

func (lp *LoggerPlugin) Name() string { return "Logger" }
func (lp *LoggerPlugin) Initialize(m *EnhancedModel) error { return nil }
func (lp *LoggerPlugin) Cleanup() error { return nil }

func (lp *LoggerPlugin) Commands() []Command {
    return []Command{
        {
            Name:        "log",
            Description: "Log custom messages",
            Usage:       "/log <message>",
            Handler:     lp.cmdLog,
        },
    }
}

func (lp *LoggerPlugin) cmdLog(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
    message := strings.Join(args, " ")
    // Custom logging logic
    return m, nil
}
```

### 2. Custom Themes

```go
// Theme system
type Theme struct {
    Primary   lipgloss.Color
    Secondary lipgloss.Color
    Success   lipgloss.Color
    Error     lipgloss.Color
    // ... more colors
}

func (t *Theme) Apply() {
    PrimaryColor = t.Primary
    SecondaryColor = t.Secondary
    SuccessColor = t.Success
    ErrorColor = t.Error
    // ... apply all colors
}

// Dark theme
var DarkTheme = &Theme{
    Primary:   lipgloss.Color("#A855F7"),
    Secondary: lipgloss.Color("#22D3EE"),
    Success:   lipgloss.Color("#34D399"),
    Error:     lipgloss.Color("#FB7185"),
}

// Light theme
var LightTheme = &Theme{
    Primary:   lipgloss.Color("#6366F1"),
    Secondary: lipgloss.Color("#06B6D4"),
    Success:   lipgloss.Color("#10B981"),
    Error:     lipgloss.Color("#EF4444"),
}

// Theme switching command
func cmdTheme(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
    if len(args) == 0 {
        m.messageQueue.Add(QueuedMessage{
            Role:      "system",
            Content:   "Available themes: dark, light",
            Timestamp: time.Now(),
            Complete:  true,
        })
        return m, nil
    }
    
    switch args[0] {
    case "dark":
        DarkTheme.Apply()
    case "light":
        LightTheme.Apply()
    default:
        m.messageQueue.Add(QueuedMessage{
            Role:      "error",
            Content:   "Unknown theme: " + args[0],
            Timestamp: time.Now(),
            Complete:  true,
        })
    }
    
    m.updateViewport()
    return m, nil
}
```

### 3. Custom Layouts

```go
// Layout manager
type LayoutManager struct {
    current LayoutType
    layouts map[LayoutType]Layout
}

type LayoutType int
const (
    LayoutDefault LayoutType = iota
    LayoutCompact
    LayoutWide
    LayoutMinimal
)

type Layout interface {
    CalculateDimensions(m *EnhancedModel) (viewportHeight, toolsHeight, processingHeight int)
    RenderSections(m *EnhancedModel) []string
}

// Compact layout implementation
type CompactLayout struct{}

func (cl *CompactLayout) CalculateDimensions(m *EnhancedModel) (int, int, int) {
    // Smaller header, no tools section, minimal processing
    return m.height - 8, 0, 1
}

func (cl *CompactLayout) RenderSections(m *EnhancedModel) []string {
    return []string{
        m.renderCompactHeader(),
        m.viewport.View(),
        m.renderDivider(),
        m.textarea.View(),
        m.renderHelpBar(),
    }
}

// Layout switching
func (m *EnhancedModel) setLayout(layout LayoutType) {
    m.layoutManager.current = layout
    m.recalculateLayout()
    m.updateViewport()
}
```

---

## Best Practices

### 1. Performance Optimization

```go
// Throttle expensive operations
type ThrottledUpdate struct {
    lastUpdate time.Time
    interval   time.Duration
}

func (tu *ThrottledUpdate) ShouldUpdate() bool {
    if time.Since(tu.lastUpdate) > tu.interval {
        tu.lastUpdate = time.Now()
        return true
    }
    return false
}

// Use in viewport updates
func (m *EnhancedModel) throttledViewportUpdate() {
    if !m.viewportThrottler.ShouldUpdate() {
        return
    }
    m.updateViewport()
}
```

### 2. Memory Management

```go
// Automatic cleanup
func (m *EnhancedModel) performMaintenance() {
    // Prune old messages
    m.messageQueue.Prune(maxQueueMessages)
    
    // Clear stream chunks
    m.messageQueue.ClearStreamChunks()
    
    // Cleanup completed tools
    m.cleanupCompletedTools()
    
    // Garbage collection hint
    runtime.GC()
}

// Schedule regular maintenance
func (m *EnhancedModel) scheduleMaintenance() tea.Cmd {
    return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
        return maintenanceMsg{}
    })
}
```

### 3. Error Handling

```go
// Centralized error handling
func (m *EnhancedModel) handleError(err error, context string) {
    m.messageQueue.Add(QueuedMessage{
        Role:      "error",
        Content:   fmt.Sprintf("Error in %s: %v", context, err),
        Timestamp: time.Now(),
        Complete:  true,
    })
    
    // Log to file
    m.agent.GetLogger().Error("TUI Error in %s: %v", context, err)
    
    m.updateViewport()
}

// Usage with recovery
func (m *EnhancedModel) safeOperation(op func() error) {
    defer func() {
        if r := recover(); r != nil {
            m.handleError(fmt.Errorf("panic: %v", r), "safe operation")
        }
    }()
    
    if err := op(); err != nil {
        m.handleError(err, "safe operation")
    }
}
```

### 4. Testing Patterns

```go
// Test helper for TUI
type TUITestHelper struct {
    model *EnhancedModel
    program *tea.Program
}

func NewTUITestHelper(agent *agent.Agent) *TUITestHelper {
    model := NewEnhancedModel(agent)
    return &TUITestHelper{model: model}
}

func (th *TUITestHelper) SendKey(key tea.KeyType) {
    msg := tea.KeyMsg{Type: key}
    th.model, _ = th.model.Update(msg)
}

func (th *TUITestHelper) SendCommand(cmd string) {
    th.model.textarea.SetValue(cmd)
    th.SendKey(tea.KeyEnter)
}

func (th *TUITestHelper) GetMessages() []QueuedMessage {
    return th.model.messageQueue.GetAll()
}

func (th *TUITestHelper) AssertLastMessage(role, content string) bool {
    messages := th.GetMessages()
    if len(messages) == 0 {
        return false
    }
    
    last := messages[len(messages)-1]
    return last.Role == role && strings.Contains(last.Content, content)
}

// Example test
func TestClearCommand(t *testing.T) {
    agent := setupTestAgent()
    helper := NewTUITestHelper(agent)
    
    // Add some messages
    helper.SendCommand("test message")
    
    // Send clear command
    helper.SendCommand("/clear")
    
    // Assert conversation is cleared
    messages := helper.GetMessages()
    if len(messages) != 1 || messages[0].Role != "system" {
        t.Error("Conversation not properly cleared")
    }
}
```

---

## Troubleshooting

### Common Issues

#### 1. Layout Problems

```go
// Debug layout calculations
func (m *EnhancedModel) debugLayout() {
    fmt.Printf("Terminal: %dx%d\n", m.width, m.height)
    fmt.Printf("Viewport: %dx%d (Y: %d)\n", m.viewport.Width, m.viewport.Height, m.viewport.YPosition)
    fmt.Printf("Tools height: %d\n", m.calculateToolsHeight())
    fmt.Printf("Processing height: %d\n", m.calculateProcessingHeight())
    fmt.Printf("Fixed height: %d\n", 2+1+m.calculateToolsHeight()+2+m.calculateProcessingHeight()+5+1)
}

// Add debug command
func cmdDebugLayout(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
    m.debugLayout()
    return m, nil
}
```

#### 2. Memory Leaks

```go
// Monitor memory usage
func (m *EnhancedModel) monitorMemory() {
    var mem runtime.MemStats
    runtime.ReadMemStats(&mem)
    
    m.messageQueue.Add(QueuedMessage{
        Role:      "system",
        Content:   fmt.Sprintf("Memory: %.2f MB (Messages: %d)", 
            float64(mem.Alloc)/1024/1024, m.messageQueue.Len()),
        Timestamp: time.Now(),
        Complete:  true,
    })
    m.updateViewport()
}
```

#### 3. Performance Issues

```go
// Profile rendering performance
func (m *EnhancedModel) profileRendering() {
    start := time.Now()
    view := m.View()
    duration := time.Since(start)
    
    fmt.Printf("Render time: %v (Length: %d)\n", duration, len(view))
}
```

### Debug Commands

```go
// Add debug commands to help troubleshoot
{
    Name:        "debug-layout",
    Category:    "Debug",
    Description: "Show layout debugging information",
    Usage:       "/debug-layout",
    Handler:     cmdDebugLayout,
},
{
    Name:        "debug-memory",
    Category:    "Debug", 
    Description: "Show memory usage information",
    Usage:       "/debug-memory",
    Handler:     cmdDebugMemory,
},
{
    Name:        "debug-render",
    Category:    "Debug",
    Description: "Profile rendering performance",
    Usage:       "/debug-render",
    Handler:     cmdDebugRender,
},
```

---

## Advanced Examples

### 1. Multi-Window Support

```go
// Window management system
type Window struct {
    ID       string
    Title    string
    Content  string
    X, Y     int
    Width    int
    Height   int
    Focused  bool
}

type WindowManager struct {
    windows  []Window
    active   string
    layout   WindowLayout
}

func (wm *WindowManager) CreateWindow(id, title string) {
    window := Window{
        ID:      id,
        Title:   title,
        X:       10,
        Y:       5,
        Width:   60,
        Height:  20,
        Focused: len(wm.windows) == 0,
    }
    wm.windows = append(wm.windows, window)
    if len(wm.windows) == 1 {
        wm.active = id
    }
}

func (wm *WindowManager) RenderWindows() string {
    var result strings.Builder
    
    for _, window := range wm.windows {
        style := lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(PrimaryColor).
            Width(window.Width).
            Height(window.Height).
            Position(window.X, window.Y)
            
        if window.Focused {
            style = style.BorderForeground(AccentColor)
        }
        
        content := fmt.Sprintf("┌ %s ┐\n", window.Title)
        content += strings.Repeat("│\n", window.Height-2)
        content += "└──────┘"
        
        result.WriteString(style.Render(content))
    }
    
    return result.String()
}
```

### 2. Real-time Collaboration

```go
// Collaboration system
type CollaborationSession struct {
    ID       string
    Users    []User
    Messages []CollaborationMessage
    Cursor   map[string]CursorPosition
}

type CollaborationMessage struct {
    UserID    string
    Content   string
    Timestamp time.Time
    Type      MessageType
}

func (m *EnhancedModel) handleCollaboration(msg CollaborationMessage) {
    // Add user indicator
    m.messageQueue.Add(QueuedMessage{
        Role:      "user",
        Content:   fmt.Sprintf("[%s] %s", msg.UserID, msg.Content),
        Timestamp: msg.Timestamp,
        Complete:  true,
    })
    
    // Update user cursors
    m.updateUserCursor(msg.UserID, msg.Cursor)
    
    m.updateViewport()
}
```

### 3. Plugin Marketplace

```go
// Plugin marketplace
type PluginMarketplace struct {
    plugins []PluginInfo
    installed map[string]bool
}

type PluginInfo struct {
    Name        string
    Version     string
    Description string
    Author      string
    URL         string
    Dependencies []string
}

func (pm *PluginMarketplace) InstallPlugin(name string) error {
    plugin := pm.findPlugin(name)
    if plugin == nil {
        return fmt.Errorf("plugin not found: %s", name)
    }
    
    // Download and install
    if err := pm.downloadPlugin(plugin); err != nil {
        return err
    }
    
    // Load and initialize
    return pm.loadPlugin(plugin)
}

func (pm *PluginMarketplace) ListPlugins() []PluginInfo {
    return pm.plugins
}
```

---

## Conclusion

This integration guide provides a comprehensive foundation for extending and customizing the ClosedWheelerAGI TUI system. The modular architecture allows for:

- **Easy Extension**: Add new commands, overlays, and features
- **Flexible Integration**: Integrate with external systems and APIs
- **Performance Optimization**: Efficient rendering and memory management
- **Robust Testing**: Comprehensive testing patterns and debugging tools
- **Modern UX**: Advanced features like multi-window support and collaboration

The TUI system is designed to be both powerful and accessible, providing a solid foundation for building sophisticated terminal-based AI interfaces.

Remember to:
1. Follow the established patterns for consistency
2. Test thoroughly with the provided testing helpers
3. Monitor performance and memory usage
4. Provide clear user feedback for all operations
5. Document custom extensions for maintainability

With these guidelines and examples, you can create powerful extensions that enhance the ClosedWheelerAGI experience while maintaining code quality and performance.
