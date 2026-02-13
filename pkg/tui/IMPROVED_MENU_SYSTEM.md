# Improved TUI Menu System Design

## Overview

This document proposes an enhanced menu system for the ClosedWheelerAGI TUI that provides better accessibility, discoverability, and ease of use compared to the current command-based approach.

## Current Issues

1. **Command Discovery**: Users must know exact command names (`/help`, `/status`, etc.)
2. **No Visual Menu**: No hierarchical menu navigation
3. **Accessibility**: Keyboard shortcuts aren't clearly visible
4. **Learning Curve**: New users must memorize commands
5. **Error-Prone**: Typos lead to "command not found" errors

## Proposed Solution: Enhanced Menu System

### 1. Quick Access Menu (F1 or Alt+M)

```
┌─ ClosedWheelerAGI Menu ─────────────────────────────────┐
│                                                        │
│ 📝 Conversation    📊 Information    📁 Project      │
│   /clear             /status           /reload         │
│   /retry             /stats            /rules          │
│   /continue          /memory           /git            │
│                      /context          /health         │
│                      /tools                            │
│                                                        │
│ ⚙️ Features         🧠 Memory & Brain  🔗 Integration │
│   /verbose           /brain            /telegram       │
│   /debug             /roadmap          /model          │
│   /timestamps        /save             /skill          │
│   /browser                            /mcp            │
│   /heartbeat                                           │
│   /pipeline                                            │
│                                                        │
│ 🤖 Dual Session     🔌 Providers       🖥️ System       │
│   /session           /providers        /logs           │
│   /debate            /pairings         /config         │
│   /conversation                       /report         │
│   /stop                               /errors         │
│                                      /resilience     │
│                                      /tool-retries   │
│                                      /retry-mode     │
│                                      /recover        │
│                                      /exit           │
│                                                        │
│ [↑↓ Navigate] [Enter Select] [Esc Close] [Tab Search] │
└────────────────────────────────────────────────────┘
```

### 2. Command Palette (Ctrl+P or Ctrl+K)

```
┌─ Command Palette ──────────────────────────────────────┐
│ > clear conversation                                   │
│                                                       │
│ 💬 /clear              Clear conversation history     │
│ 💬 /retry              Retry last message             │
│ 💬 /continue           Continue last response         │
│ 📊 /status             Show system status             │
│ 📊 /stats              Show API usage statistics      │
│ 📁 /reload             Reload project files           │
│ ⚙️ /verbose on         Toggle verbose mode            │
│                                                       │
│ Type to filter commands...                           │
│                                                       │
│ [↑↓ Navigate] [Enter Execute] [Esc Close]           │
└─────────────────────────────────────────────────────┘
```

### 3. Context Menu (Right-click or Ctrl+X)

```
┌─ Context Menu ────────────────────────────────────────┐
│                                                       │
│ 📋 Copy Selection                                     │
│ ✂️ Cut Selection                                      │
│ 📄 Paste                                              │
│ ──────────────────────────────                        │
│ 🔍 Search in conversation...                          │
│ 🏃 Jump to bottom                                     │
│ 📜 Show history                                       │
│ ──────────────────────────────                        │
│ 💾 Save conversation                                  │
│ 📤 Export conversation                                │
│ 🗑️ Clear conversation                                 │
│                                                       │
└─────────────────────────────────────────────────────┘
```

### 4. Help System (F1 or ?)

```
┌─ Help & Documentation ────────────────────────────────┐
│                                                       │
│ 📖 Getting Started                                    │
│   • Basic usage                                       │
│   • Command reference                                │
│   • Keyboard shortcuts                               │
│                                                       │
│ ⌨️ Quick Reference                                    │
│   Ctrl+C    Stop current request                      │
│   Ctrl+P    Open command palette                      │
│   Ctrl+K    Open command palette                      │
│   F1        Show this help                            │
│   Alt+M     Open main menu                            │
│   Tab       Next input field                          │
│   Shift+Tab Previous input field                     │
│   Esc       Close dialog/Cancel                       │
│                                                       │
│ 🎯 Common Tasks                                       │
│   • Start new conversation                            │
│   • Change model/provider                            │
│   • View system status                                │
│   • Configure settings                                │
│                                                       │
│ 🔍 Search help...                                     │
│                                                       │
│ [Tab Navigate] [Enter Select] [Esc Close]            │
└─────────────────────────────────────────────────────┘
```

## Implementation Plan

### Phase 1: Core Menu Infrastructure

```go
// New menu system types
type MenuType int
const (
    MenuMain MenuType = iota
    MenuCommandPalette
    MenuContext
    MenuHelp
)

type MenuItem struct {
    Icon        string
    Title       string
    Description string
    Command     string
    Category    string
    Shortcut    string
}

type MenuState struct {
    Active      bool
    Type        MenuType
    Cursor      int
    Search      string
    Items       []MenuItem
    Filtered    []MenuItem
}
```

### Phase 2: Menu Rendering

```go
// Add to EnhancedModel
menuState MenuState

// Add to View() method
if m.menuState.Active {
    return m.renderMenu()
}

// Menu rendering function
func (m *EnhancedModel) renderMenu() string {
    switch m.menuState.Type {
    case MenuMain:
        return m.renderMainMenu()
    case MenuCommandPalette:
        return m.renderCommandPalette()
    case MenuContext:
        return m.renderContextMenu()
    case MenuHelp:
        return m.renderHelpMenu()
    }
}
```

### Phase 3: Keyboard Integration

```go
// Add to Update() method
case tea.KeyMsg:
    switch msg.Type {
    case tea.KeyCtrlP, tea.KeyCtrlK:
        m.openMenu(MenuCommandPalette)
        return m, nil
    case tea.KeyF1:
        if m.menuState.Active {
            m.closeMenu()
        } else {
            m.openMenu(MenuHelp)
        }
        return m, nil
    case tea.KeyAltM:
        m.openMenu(MenuMain)
        return m, nil
    }
```

### Phase 4: Search and Filtering

```go
func (m *EnhancedModel) filterMenuItems(search string) {
    if search == "" {
        m.menuState.Filtered = m.menuState.Items
        return
    }
    
    search = strings.ToLower(search)
    var filtered []MenuItem
    
    for _, item := range m.menuState.Items {
        if strings.Contains(strings.ToLower(item.Title), search) ||
           strings.Contains(strings.ToLower(item.Description), search) ||
           strings.Contains(strings.ToLower(item.Command), search) {
            filtered = append(filtered, item)
        }
    }
    
    m.menuState.Filtered = filtered
}
```

## Enhanced Features

### 1. Command Discovery

- **Fuzzy Search**: Find commands by typing partial names
- **Category Filtering**: Show commands from specific categories
- **Recent Commands**: Quick access to recently used commands
- **Favorite Commands**: Pin frequently used commands

### 2. Accessibility Improvements

- **Keyboard Navigation**: Full keyboard accessibility
- **Screen Reader Support**: Proper ARIA labels and descriptions
- **High Contrast Mode**: Enhanced visual clarity
- **Large Text Mode**: Increased font size option

### 3. Visual Enhancements

- **Icons and Emojis**: Visual cues for different command types
- **Color Coding**: Consistent color scheme for categories
- **Animations**: Smooth transitions and micro-interactions
- **Tooltips**: Hover hints for keyboard shortcuts

### 4. Smart Features

- **Context Awareness**: Show relevant commands based on current state
- **Learning System**: Suggest commands based on usage patterns
- **Command History**: Navigate through previous commands
- **Bulk Operations**: Execute multiple commands in sequence

## Migration Strategy

### Step 1: Backward Compatibility

- Keep all existing slash commands working
- Add new menu system alongside current system
- Provide migration guide for users

### Step 2: Gradual Adoption

- Introduce menu system as optional enhancement
- Add tooltips showing menu equivalents for slash commands
- Collect user feedback and refine experience

### Step 3: Default Experience

- Make menu system the primary interaction method
- Keep slash commands for power users
- Provide comprehensive documentation

## Code Examples

### Menu Item Definition

```go
var menuItems = []MenuItem{
    {
        Icon:        "💬",
        Title:       "Clear Conversation",
        Description: "Clear all conversation history",
        Command:     "/clear",
        Category:    "Conversation",
        Shortcut:    "Ctrl+L",
    },
    {
        Icon:        "📊",
        Title:       "System Status",
        Description: "Show detailed system and project status",
        Command:     "/status detailed",
        Category:    "Information",
        Shortcut:    "Ctrl+S",
    },
    // ... more items
}
```

### Menu Event Handling

```go
func (m *EnhancedModel) handleMenuSelection(item MenuItem) (tea.Model, tea.Cmd) {
    // Close menu
    m.closeMenu()
    
    // Execute command
    if cmd := FindCommand(strings.TrimPrefix(item.Command, "/")); cmd != nil {
        return cmd.Handler(m, strings.Fields(item.Command)[1:])
    }
    
    return m, nil
}
```

### Search Implementation

```go
func (m *EnhancedModel) updateMenuSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.Type {
    case tea.KeyBackspace:
        if len(m.menuState.Search) > 0 {
            m.menuState.Search = m.menuState.Search[:len(m.menuState.Search)-1]
        }
    case tea.KeyRunes:
        m.menuState.Search += string(msg.Runes)
    default:
        return m, nil
    }
    
    m.filterMenuItems(m.menuState.Search)
    m.menuState.Cursor = 0
    
    return m, nil
}
```

## Benefits

1. **Improved Discoverability**: Users can browse all available commands
2. **Better Accessibility**: Full keyboard navigation and screen reader support
3. **Reduced Learning Curve**: Visual menu vs memorizing commands
4. **Enhanced Productivity**: Quick access via keyboard shortcuts
5. **Modern UX**: Familiar menu paradigms from modern applications
6. **Error Reduction**: No more "command not found" errors
7. **Context Awareness**: Show relevant commands based on current state

## Conclusion

The enhanced menu system provides a significant improvement in user experience while maintaining backward compatibility with existing slash commands. The phased implementation approach ensures smooth migration and allows for iterative refinement based on user feedback.

This design makes ClosedWheelerAGI more accessible to new users while providing power users with efficient keyboard shortcuts and advanced features.
