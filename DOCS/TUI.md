# TUI Function Documentation - ClosedWheelerAGI

## Table of Contents
1. [Core TUI Functions](#core-tui-functions)
2. [Message Queue System](#message-queue-system)
3. [Command System](#command-system)
4. [Rendering Functions](#rendering-functions)
5. [Layout Management](#layout-management)
6. [State Management](#state-management)
7. [Animation System](#animation-system)
8. [Integration Examples](#integration-examples)
9. [Best Practices](#best-practices)

---

## Core TUI Functions

### `NewEnhancedModel(ag *agent.Agent) *EnhancedModel`

**Purpose**: Creates and initializes the main TUI model with all UI components.

**Parameters**:
- `ag *agent.Agent`: The agent instance that will power the TUI

**Returns**:
- `*EnhancedModel`: A fully initialized TUI model ready to run

**Example Usage**:
```go
// Initialize agent
agent := agent.New(config)

// Create TUI model
model := tui.NewEnhancedModel(agent)

// Run the TUI
program := tea.NewProgram(model)
if _, err := program.Run(); err != nil {
    log.Fatal(err)
}
```

**Initialization Details**:
- Sets up text input with placeholder "Message ClosedWheelerAGI..."
- Configures spinner with dot animation
- Creates progress bar with gradient styling
- Initializes help system
- Sets up confirm dialog, log table, and conversation paginator
- Loads welcome message into message queue
- Initializes provider manager and retry wrapper
- Creates dual session for agent-to-agent conversations

---

### `Init() tea.Cmd`

**Purpose**: Initializes the TUI with startup animations and commands.

**Parameters**: None (method on EnhancedModel)

**Returns**:
- `tea.Cmd`: Bubble Tea command sequence for startup

**Example Usage**:
```go
// This is called automatically by Bubble Tea
// No direct usage needed, but understanding helps with debugging
func (m *EnhancedModel) Init() tea.Cmd {
    return tea.Sequence(
        textarea.Blink,  // Start cursor blinking
        m.spinner.Tick,  // Start spinner animation
        tickAnimation(), // Start animation ticker
    )
}
```

**Startup Sequence**:
1. Cursor begins blinking in input area
2. Spinner starts rotating for loading states
3. Animation ticker begins (4 FPS, 250ms intervals)

---

### `Update(msg tea.Msg) (tea.Model, tea.Cmd)`

**Purpose**: Main event handler that processes all user interactions and system events.

**Parameters**:
- `msg tea.Msg`: Any Bubble Tea message (key presses, window resizes, custom messages)

**Returns**:
- `tea.Model`: Updated model state
- `tea.Cmd`: Command to execute

**Example Usage**:
```go
// Handle custom message
case myCustomMsg:
    // Update model state
    m.status = "Custom event received"
    // Return command to execute
    return m, func() tea.Msg {
        return anotherCustomMsg{}
    }
```

**Event Types Handled**:
- **Key Events**: Ctrl+C (quit/stop), Enter (send), Esc (stop), navigation keys
- **Window Events**: Resize handling and layout recalculation
- **Custom Events**: Response completion, streaming chunks, tool execution, pipeline status
- **Overlay Events**: Help menu, settings, panels, debate wizard

**Key Features**:
- Intercepts keys when overlays are active
- Handles message queuing during processing
- Manages tool execution state
- Updates viewport and layout
- Handles conversation streaming

---

### `View() string`

**Purpose**: Renders the complete TUI interface as a string.

**Parameters**: None (method on EnhancedModel)

**Returns**:
- `string`: Complete rendered TUI ready for display

**Example Usage**:
```go
// This is called automatically by Bubble Tea
// The rendering order is important for proper layout:
func (m *EnhancedModel) View() string {
    if !m.ready {
        return m.spinner.View() + " Initializing ClosedWheelerAGI..."
    }

    // Check for active overlays (rendered first)
    if m.pickerActive { return m.enhancedPickerView() }
    if m.helpActive { return m.helpMenuView() }
    if m.panelActive { return m.panelView() }
    if m.settingsActive { return m.settingsView() }
    if m.debateWizActive { return m.debateWizardView() }
    if m.debateViewActive { return m.debateViewerView() }

    // Build main interface sections
    sections := []string{
        m.renderHeader(),      // App title and version
        m.renderStatusBar(),   // Status indicators and stats
        m.renderActiveTools(), // Active tools (when not processing)
        m.renderDivider(),     // Visual separator
        m.viewport.View(),     // Main conversation area
        m.renderDivider(),     // Visual separator
        m.renderProcessingArea(), // Processing indicator (when active)
        m.textarea.View(),     // Input area (always visible)
        m.renderHelpBar(),     // Help shortcuts
    }

    return strings.Join(sections, "\n")
}
```

**Rendering Priority**:
1. Loading screen (if not ready)
2. Active overlays (picker, help, panel, settings, debate)
3. Main interface sections in order

---

## Message Queue System

### `NewMessageQueue() *MessageQueue`

**Purpose**: Creates a thread-safe message queue for conversation management.

**Parameters**: None

**Returns**:
- `*MessageQueue`: New empty message queue

**Example Usage**:
```go
// Create new message queue
mq := NewMessageQueue()

// Add welcome message
mq.Add(QueuedMessage{
    Role:      "system",
    Content:   "Welcome to ClosedWheelerAGI!",
    Timestamp: time.Now(),
    Complete:  true,
})
```

---

### `Add(msg QueuedMessage)`

**Purpose**: Thread-safely adds a message to the queue.

**Parameters**:
- `msg QueuedMessage`: Message to add

**Returns**: None

**Example Usage**:
```go
// Add user message
mq.Add(QueuedMessage{
    Role:      "user",
    Content:   "Hello, how are you?",
    Timestamp: time.Now(),
    Complete:  true,
})

// Add streaming assistant message
mq.Add(QueuedMessage{
    Role:      "assistant",
    Content:   "",
    Streaming: true,
    Timestamp: time.Now(),
    Complete:  false,
})
```

**Thread Safety**: Uses write mutex to ensure safe concurrent access.

---

### `UpdateLast(update func(*QueuedMessage))`

**Purpose**: Updates the most recent message in the queue.

**Parameters**:
- `update func(*QueuedMessage)`: Function that modifies the message

**Returns**: None

**Example Usage**:
```go
// Update streaming message content
mq.UpdateLast(func(qm *QueuedMessage) {
    qm.Content += "New chunk of text"
    qm.StreamChunk += "New chunk"
})

// Mark message as complete
mq.UpdateLast(func(qm *QueuedMessage) {
    qm.Complete = true
    qm.Streaming = false
    qm.Stats = &MessageStats{
        PromptTokens:     100,
        CompletionTokens: 50,
        Elapsed:          2 * time.Second,
    }
})
```

**Use Cases**:
- Streaming response updates
- Adding thinking/reasoning content
- Marking messages complete
- Adding tool execution results

---

### `GetAll() []QueuedMessage`

**Purpose**: Returns a thread-safe copy of all messages.

**Parameters**: None

**Returns**:
- `[]QueuedMessage`: Copy of all messages in queue

**Example Usage**:
```go
// Get all messages for rendering
messages := mq.GetAll()

// Find last user message
var lastUserMsg string
for i := len(messages) - 1; i >= 0; i-- {
    if messages[i].Role == "user" {
        lastUserMsg = messages[i].Content
        break
    }
}
```

**Thread Safety**: Uses read mutex and creates a deep copy to prevent data races.

---

### `Clear()`

**Purpose**: Removes all messages from the queue.

**Parameters**: None

**Returns**: None

**Example Usage**:
```go
// Clear conversation
mq.Clear()

// Add system message after clear
mq.Add(QueuedMessage{
    Role:      "system",
    Content:   "✨ Conversation cleared.",
    Timestamp: time.Now(),
    Complete:  true,
})
```

---

### `Prune(maxMessages int)`

**Purpose**: Removes old messages to prevent memory growth.

**Parameters**:
- `maxMessages int`: Maximum number of messages to keep

**Returns**: None

**Example Usage**:
```go
// Keep only last 200 messages
mq.Prune(200)

// Keep last 50 messages for memory efficiency
mq.Prune(50)
```

**Memory Management**: Automatically called after response completion with `maxQueueMessages` (200).

---

### `Len() int`

**Purpose**: Returns current number of messages in queue.

**Parameters**: None

**Returns**:
- `int`: Number of messages

**Example Usage**:
```go
// Check if queue is empty
if mq.Len() == 0 {
    fmt.Println("No messages in queue")
}

// Log queue size periodically
fmt.Printf("Queue size: %d messages\n", mq.Len())
```

---

### `ClearStreamChunks()`

**Purpose**: Removes duplicate stream chunk data to reclaim memory.

**Parameters**: None

**Returns**: None

**Example Usage**:
```go
// Clean up after streaming is complete
mq.ClearStreamChunks()
```

**Memory Optimization**: Stream chunks are stored alongside content during streaming for performance, but cleared after completion to save memory.

---

## Command System

### `GetAllCommands() []CommandCategory`

**Purpose**: Returns all available commands organized by category.

**Parameters**: None

**Returns**:
- `[]CommandCategory`: All commands in categorized structure

**Example Usage**:
```go
// Get all commands for help system
categories := GetAllCommands()

// Display all commands
for _, category := range categories {
    fmt.Printf("%s %s\n", category.Icon, category.Name)
    for _, cmd := range category.Commands {
        fmt.Printf("  /%s - %s\n", cmd.Name, cmd.Description)
    }
}
```

**Command Categories**:
1. **Conversation** (💬): clear, retry, continue
2. **Information** (📊): status, stats, memory, context, tools
3. **Project** (📁): reload, rules, git, health
4. **Features** (⚙️): verbose, debug, timestamps, browser, heartbeat, pipeline
5. **Memory & Brain** (🧠): brain, roadmap, save
6. **Integration** (🔗): telegram, model, skill, mcp
7. **Providers** (🔌): providers, pairings
8. **Dual Session** (🤖): session, debate, conversation, stop
9. **System** (🖥️): logs, config, report, errors, resilience, tool-retries, retry-mode, recover, help, exit
10. **Interface** (🖥️): logs, history

---

### `FindCommand(name string) *Command`

**Purpose**: Finds a command by name or alias.

**Parameters**:
- `name string`: Command name or alias (with or without /)

**Returns**:
- `*Command`: Command pointer or nil if not found

**Example Usage**:
```go
// Find command by name
cmd := FindCommand("clear")
if cmd != nil {
    fmt.Printf("Found: %s - %s\n", cmd.Name, cmd.Description)
}

// Find by alias
cmd = FindCommand("c") // alias for clear
if cmd != nil {
    fmt.Printf("Alias for: %s\n", cmd.Name)
}

// Find with slash
cmd = FindCommand("/status")
if cmd != nil {
    fmt.Printf("Command: %s\n", cmd.Usage)
}
```

**Search Behavior**:
- Case-insensitive
- Accepts names with or without leading slash
- Searches both primary names and aliases

---

### Command Handlers Pattern

All command handlers follow this signature:
```go
func cmd<Name>(m *EnhancedModel, args []string) (tea.Model, tea.Cmd)
```

**Example Implementation**:
```go
func cmdClear(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
    // Clear message queue
    m.messageQueue.Clear()
    
    // Add system message
    m.messageQueue.Add(QueuedMessage{
        Role:      "system",
        Content:   "✨ Conversation cleared.",
        Timestamp: time.Now(),
        Complete:  true,
    })
    
    // Update viewport
    m.updateViewport()
    
    // Return updated model (no command)
    return m, nil
}
```

**Common Patterns**:
1. **Validation**: Check arguments and show errors if invalid
2. **State Updates**: Modify model state as needed
3. **Feedback**: Add messages to queue for user feedback
4. **Viewport Updates**: Call `m.updateViewport()` to refresh display
5. **Return**: Return updated model and optional command

---

## Rendering Functions

### `renderHeader() string`

**Purpose**: Renders the top header with app title, version, and model info.

**Parameters**: None (method on EnhancedModel)

**Returns**:
- `string`: Rendered header

**Example Output**:
```
◈ CLOSED WHEELER AGI  v2.1                    λ gpt-4
```

**Layout**:
- Left: App title with icon
- Center: Version number
- Right: Current model badge

---

### `renderStatusBar() string`

**Purpose**: Renders status bar with activity badge and system statistics.

**Parameters**: None (method on EnhancedModel)

**Returns**:
- `string`: Rendered status bar

**Example Output**:
```
● WORKING    STM:15 WM:8 CTX:42 TOK:1.2K
```

**Status Indicators**:
- **IDLE**: No active processing
- **THINKING**: Agent is reasoning
- **WORKING**: Tool is executing

**Statistics**:
- **STM**: Short-term memory items
- **WM**: Working memory items
- **CTX**: Context message count
- **TOK**: Total tokens (formatted)

---

### `renderActiveTools() string`

**Purpose**: Renders currently active tools with status indicators.

**Parameters**: None (method on EnhancedModel)

**Returns**:
- `string`: Rendered tools section

**Example Output**:
```
🔧 ✓ read_file (125ms) │ ○ write_file │ ● browser_navigate
```

**Status Icons**:
- **●**: Currently running (with spinner)
- **✓**: Completed successfully
- **✗**: Failed
- **○**: Pending

**Display**: Shows last 3 tools to prevent layout overflow.

---

### `renderProcessingArea() string`

**Purpose**: Renders processing indicator above input during active operations.

**Parameters**: None (method on EnhancedModel)

**Returns**:
- `string`: Rendered processing area (exactly 2 lines)

**Example Output**:
```
● WORKING: read_file.... [2.3s]
↳ /path/to/file.txt
```

**Content**:
- Line 1: Status + tool name + elapsed time
- Line 2: Pipeline bar OR tool argument OR thinking preview

---

### `renderHelpBar() string`

**Purpose**: Renders bottom help bar with keyboard shortcuts.

**Parameters**: None (method on EnhancedModel)

**Returns**:
- `string`: Rendered help bar

**Example Output**:
```
↵ Send │ /help Commands │ ^C Quit
```

**Context-sensitive**:
- Normal mode: Send/help/quit shortcuts
- Processing mode: Stop request shortcuts

---

### `renderDivider() string`

**Purpose**: Renders visual separator between sections.

**Parameters**: None (method on EnhancedModel)

**Returns**:
- `string`: Rendered divider line

**Example Output**:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Styling**: Uses heavy horizontal line with indigo glow effect.

---

## Layout Management

### `calculateViewportHeight() int`

**Purpose**: Calculates optimal viewport height based on terminal size and fixed components.

**Parameters**: None (method on EnhancedModel)

**Returns**:
- `int`: Viewport height in lines

**Calculation**:
```go
fixedHeight := headerH + statusH + toolsH + dividers + processingH + inputH + helpH
viewportHeight := m.height - fixedHeight
if viewportHeight < 5 {
    viewportHeight = 5  // Minimum height
}
```

**Components Considered**:
- Header: 2 lines
- Status bar: 1 line
- Tools: 0-1 lines (dynamic)
- Dividers: 2 lines
- Processing: 0-2 lines (dynamic)
- Input: 5 lines (fixed)
- Help bar: 1 line

---

### `recalculateLayout()`

**Purpose**: Recalculates all layout dimensions when terminal size or content changes.

**Parameters**: None (method on EnhancedModel)

**Returns**: None

**Example Usage**:
```go
// Called automatically on window resize
func (m *EnhancedModel) recalculateLayout() {
    viewportHeight := m.calculateViewportHeight()
    vpWidth := m.width - 2
    
    // Calculate Y position (rows above viewport)
    toolsH := m.calculateToolsHeight()
    yPosition := 1 + 1 + toolsH + 1  // = 3 + toolsH
    
    // Update viewport dimensions
    m.viewport.Width = vpWidth
    m.viewport.Height = viewportHeight
    m.viewport.YPosition = yPosition
    
    // Update textarea width
    m.textarea.SetWidth(m.width - 8)
}
```

**Trigger Events**:
- Terminal window resize
- Tools become active/inactive
- Processing starts/stops

---

## State Management

### `SetState(s TUIState)`

**Purpose**: Updates the current TUI state (which overlay is active).

**Parameters**:
- `s TUIState`: New state to set

**Returns**: None

**Available States**:
```go
const (
    StateMain TUIState = iota          // Main chat interface
    StateModelPicker                   // Model selection overlay
    StateHelpMenu                      // Help menu overlay
    StateInfoPanel                     // Information panel overlay
    StateSettings                      // Settings overlay
    StateDebateWizard                  // Debate setup wizard
    StateDebateViewer                  // Debate viewing overlay
)
```

**Example Usage**:
```go
// Switch to help menu
m.SetState(StateHelpMenu)
m.helpActive = true

// Return to main interface
m.SetState(StateMain)
m.helpActive = false
```

---

### `GetState() TUIState`

**Purpose**: Returns the current TUI state.

**Parameters**: None (method on EnhancedModel)

**Returns**:
- `TUIState`: Current state

**Example Usage**:
```go
// Check current state
switch m.GetState() {
case StateMain:
    // Handle main interface logic
case StateHelpMenu:
    // Handle help menu logic
case StateSettings:
    // Handle settings logic
}
```

---

## Animation System

### `tickAnimation() tea.Cmd`

**Purpose**: Creates periodic animation command for smooth UI animations.

**Parameters**: None

**Returns**:
- `tea.Cmd`: Command that emits animation ticks

**Example Usage**:
```go
// This is called in Init() and handled in Update()
func tickAnimation() tea.Cmd {
    return tea.Tick(time.Millisecond*250, func(t time.Time) tea.Msg {
        return animationTickMsg{}
    })
}

// Handle in Update()
case animationTickMsg:
    m.thinkingAnimation = (m.thinkingAnimation + 1) % 4
    m.updateViewport()
    return m, tickAnimation()  // Schedule next tick
```

**Animation Details**:
- **Interval**: 250ms (4 FPS)
- **Used for**: Thinking dots, spinner updates, status animations
- **Automatic**: Continuously runs while TUI is active

---

## Integration Examples

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
    // Load configuration
    config, err := agent.LoadConfig("config.json")
    if err != nil {
        log.Fatal(err)
    }
    
    // Initialize agent
    ag, err := agent.New(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create TUI model
    model := tui.NewEnhancedModel(ag)
    
    // Run the program
    program := tea.NewProgram(model, tea.WithAltScreen())
    if _, err := program.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Command Implementation

```go
// Add custom command to GetAllCommands()
{
    Name:        "custom",
    Aliases:     []string{"cst"},
    Category:    "Custom",
    Description: "Custom command example",
    Usage:       "/custom [arg1] [arg2]",
    Handler:     cmdCustom,
}

// Implement handler
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
    
    // Process command
    result := fmt.Sprintf("Processed: %s + %s", args[0], args[1])
    
    // Add feedback
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

### Custom Overlay Implementation

```go
// Add to EnhancedModel struct
customOverlayActive bool
customOverlayCursor int

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
                m.customOverlayCursor = len(m.customItems) - 1
            }
            return m, nil
        case tea.KeyDown:
            m.customOverlayCursor = (m.customOverlayCursor + 1) % len(m.customItems)
            return m, nil
        }
    }
}

// Implement render method
func (m *EnhancedModel) renderCustomOverlay() string {
    // Custom overlay rendering logic
    return "Custom Overlay Content"
}
```

---

## Best Practices

### 1. Thread Safety
- Always use message queue methods for thread-safe operations
- Never modify message queue directly
- Use `UpdateLast()` for streaming updates

### 2. Memory Management
- Call `Prune()` regularly to prevent memory growth
- Use `ClearStreamChunks()` after streaming completion
- Limit displayed content to prevent viewport issues

### 3. Performance
- Throttle viewport updates during streaming (50ms intervals)
- Use layout caching for expensive calculations
- Minimize string allocations in hot paths

### 4. User Experience
- Always provide feedback for user actions
- Use consistent styling and terminology
- Handle edge cases gracefully

### 5. Code Organization
- Keep command handlers focused and small
- Use utility functions for common operations
- Document complex rendering logic

### 6. Error Handling
- Validate inputs before processing
- Show clear error messages to users
- Log technical errors for debugging

### 7. Testing
- Test command handlers with various inputs
- Verify layout calculations with different terminal sizes
- Test message queue thread safety

---

## Advanced Usage

### Custom Message Types

```go
// Define custom message
type customProgressMsg struct {
    percent int
    message string
}

// Send custom command
return m, func() tea.Msg {
    return customProgressMsg{percent: 50, message: "Processing..."}
}

// Handle in Update()
case customProgressMsg:
    // Update progress
    return m, nil
```

### Dynamic Layout Updates

```go
// Trigger layout recalculation
func (m *EnhancedModel) forceLayoutUpdate() {
    m.recalculateLayout()
    m.updateViewport()
}

// Handle dynamic content
if contentChanged {
    m.forceLayoutUpdate()
}
```

### Custom Styling

```go
// Add custom styles to styles.go
var (
    CustomStyle = lipgloss.NewStyle().
        Foreground(PrimaryColor).
        Bold(true).
        Padding(0, 1)
)

// Use in rendering
content := CustomStyle.Render("Custom styled text")
```

This comprehensive documentation provides a complete reference for all TUI functions, their usage patterns, and integration examples. Use this as a guide for extending, maintaining, or understanding the ClosedWheelerAGI TUI system.
