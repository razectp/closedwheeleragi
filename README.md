# ClosedWheelerAGI

An advanced AI-powered terminal interface with multi-agent capabilities, intelligent tool execution, and comprehensive conversation management.

## 🚀 Features

- **Multi-Agent Pipeline**: Planner → Researcher → Executor → Critic workflow
- **Intelligent Tool Retry**: Automatic retry with exponential backoff and error analysis
- **Real-time Streaming**: Live conversation updates with thinking/reasoning display
- **Provider Management**: Support for multiple LLM providers with seamless switching
- **Dual Session Mode**: Agent-to-agent debates and conversations
- **Comprehensive TUI**: Modern terminal interface with help system and overlays
- **Memory Management**: Tiered memory system (short-term, working, long-term)
- **Error Recovery**: Resilient error handling with automatic recovery procedures

## 📋 Quick Start

### Prerequisites

- Go 1.21 or later
- Git

### Installation

```bash
git clone https://github.com/razectp/closedwheeleragi.git
cd closedwheeleragi
go mod tidy
go build -o closedwheeleragi cmd/agi/main.go
```

### Configuration

1. Copy the example configuration:
```bash
cp config.json.example config.json
```

2. Edit `config.json` with your API keys and preferences:

```json
{
  "model": "gpt-4",
  "api_key": "your-api-key-here",
  "ui": {
    "verbose": true,
    "timestamps": true
  }
}
```

### Running

```bash
./closedwheeleragi
```

## 🎯 Basic Usage

### TUI Commands

The TUI supports slash commands for various operations:

- **`/help`** - Show available commands
- **`/clear`** - Clear conversation history  
- **`/status`** - Show system status
- **`/model`** - Change AI model/provider
- **`/debate`** - Start agent debate
- **`/providers`** - Manage LLM providers

### Keyboard Shortcuts

- **`Ctrl+C`** - Stop current request / Quit
- **`Enter`** - Send message
- **`Esc`** - Cancel operation
- **`F1`** - Show help menu
- **`Ctrl+P`** - Open command palette (planned)

## 📚 Documentation

Comprehensive documentation is available in the `DOCS/` folder:

- **[TUI Documentation](DOCS/TUI.md)** - Complete TUI function reference
- **[Integration Guide](DOCS/INTEGRATION.md)** - Developer integration guide
- **[Menu System Design](DOCS/MENU_SYSTEM.md)** - Improved menu system proposal

## 🏗️ Architecture

```
ClosedWheelerAGI/
├── cmd/agi/           # Main application entry point
├── pkg/               # Core packages
│   ├── agent/         # AI agent implementation
│   ├── tui/           # Terminal user interface
│   ├── providers/     # LLM provider management
│   ├── tools/         # Tool execution system
│   └── config/        # Configuration management
├── DOCS/              # Documentation
└── config.json        # Configuration file
```

## 🔧 Configuration

### Main Configuration Options

```json
{
  "model": "gpt-4",
  "api_key": "your-api-key",
  "ui": {
    "verbose": false,
    "timestamps": true,
    "debug_tools": false
  },
  "browser": {
    "headless": true,
    "slowmo": 0
  },
  "heartbeat_interval": 30,
  "pipeline_enabled": false,
  "providers": [],
  "mcp_servers": []
}
```

## 🤖 Multi-Agent System

The multi-agent pipeline enables complex task decomposition:

1. **Planner** - Breaks down tasks into steps
2. **Researcher** - Gathers information and context
3. **Executor** - Implements solutions using tools
4. **Critic** - Reviews and refines results

Enable with: `/pipeline on`

## 🛠️ Tools

Built-in tool capabilities:

- **File Operations**: Read, write, edit files
- **Browser Automation**: Web navigation and interaction
- **Git Operations**: Version control tasks
- **Code Analysis**: Security scanning and diagnostics
- **Task Management**: Todo list and project tracking

## 🔍 Debugging

Enable debug mode for detailed logging:

```bash
# Enable verbose mode
/verbose on

# Enable tool debugging
/debug on

# Show system status
/status detailed
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI
- Inspired by modern AI assistant interfaces
- Thanks to all contributors and the open-source community

## 📞 Support

- 📖 Check the [DOCS/](DOCS/) folder for detailed documentation
- 🐛 Report issues on [GitHub Issues](https://github.com/razectp/closedwheeleragi/issues)
- 💬 Join discussions in [GitHub Discussions](https://github.com/razectp/closedwheeleragi/discussions)

---

**ClosedWheelerAGI** - Advanced AI Terminal Interface
