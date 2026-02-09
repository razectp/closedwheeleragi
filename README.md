# ClosedWheelerAGI 🛞🔒

Hi, I'm **ClosedWheelerAGI** — the fully open-source AGI that's ironically named "Closed".  
Don't worry, I'm 100% open source... just don't ask me to open any doors with leaked credentials.  
(Not yet, anyway... 😏)

Version 2.0 | Vibecoded by Cezar Trainotti Paiva

[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT)

---

## ✨ What Is This?

ClosedWheeler AGI is an intelligent coding assistant that helps you build, debug, and understand code. It features advanced context optimization, browser automation, and self-configuring AI models.

## 🚀 Quick Start

### 1. First Time Setup

```bash
# Run the agent
.\ClosedWheeler.exe

# Follow the interactive setup wizard:
# - Name your agent
# - Configure API (OpenAI/Anthropic/Local)
# - Let the model configure itself! (NEW!)
# - Choose permissions
# - Select rules preset
# - Optional: Telegram integration
```

### 2. Daily Use

```bash
# Start the agent
.\ClosedWheeler.exe

# Available commands:
/help       - Show all commands
/model      - Switch models
/config reload - Reload configuration
/clear      - Clear conversation
```

---

## 🎯 Key Features

### 🎤 **Self-Configuring Models** (NEW!)
Models interview themselves and configure optimal parameters automatically.
- **Zero manual config** - Just provide API key
- **Accurate** - Model knows itself best
- **Future-proof** - Works with any new model

### 🚀 **Context Optimization**
Smart caching system that saves 60-80% on tokens.
- **First message**: Sends full context
- **Next messages**: Only new content
- **Auto-compression**: When context grows
- **Result**: 2-3x faster, 3x more messages

### 🌐 **Browser Automation**
Navigate the web with Playwright integration.
- 9 tools for complete browser control
- AI-optimized screenshots
- Element mapping with coordinates
- Task-specific tab management

### 🔄 **Fallback Models**
Automatic failover to backup models if primary is slow.
- Zero context loss
- Configurable timeout
- Transparent logging

### 💬 **Telegram Integration**
Control your agent from Telegram.
- Full conversation support
- Approval workflow for sensitive actions
- Admin commands

### 🧠 **Brain & Roadmap** (NEW!)
Transparent learning and strategic planning.
- **Brain** (`workplace/brain.md`) - Visible knowledge base
- **Roadmap** (`workplace/roadmap.md`) - Strategic objectives
- **Health Check** - Automated project monitoring
- **Deep Reflection** - Strategic analysis every 5 heartbeats

---

## 🏗️ Project Structure

```
ClosedWheelerAGI/
├── ClosedWheeler.exe       # Main executable (13MB)
├── README.md               # This file
├── .agi/                   # Runtime data
│   ├── config.json         # Configuration
│   ├── memory.json         # Long-term memory
│   └── logs/               # Log files
├── workplace/              # Your workspace
│   └── .agirules           # Agent rules
├── docs/                   # Technical documentation
└── pkg/                    # Source code
```

---

## 📖 Documentation

- **[Quick Guides](docs/guides/ENHANCED_SETUP_GUIDE.md)** - Comprehensive setup guide
- **[Strategic Planning](docs/guides/BRAIN_ROADMAP_HEALTH_GUIDE.md)** - Guide to Brain, Roadmap, and Health systems
- **[Technical Index](docs/README.md)** - Deep dives into all major features

---

## 🐛 Troubleshooting

Check logs in `.agi/logs/latest.log` or use `/config reload` if configuration feels stale.

---

## 🙏 Credits

**Vibecoded by**: Cezar Trainotti Paiva

---

## ☕ Support & Donations

If you find this project useful and want to support its development, you can donate via:

- **Bitcoin (BTC)**: `bc1px38hyrc4kufzxdz9207rsy5cn0hau2tfhf3678wz3uv9fpn2m0msre98w7`
- **Solana (SOL)**: `3pPpEcGEmtjCYokm8sRUu6jzjjkmfpv3qnz2pGdVYnKH`
- **Ethereum (ETH)**: `0xF465cc2d41b2AA66393ae110396263C20746CfC9`

Your support helps keep the code flowing and the agent evolving! 🦅
