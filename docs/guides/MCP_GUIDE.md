# MCP Integration Guide

ClosedWheelerAGI supports the **Model Context Protocol (MCP)** for connecting to external tool servers. MCP allows the agent to discover and use tools provided by any MCP-compatible server, greatly extending its capabilities.

## What is MCP?

MCP (Model Context Protocol) is an open standard for connecting AI assistants to external tools and data sources. It defines a protocol for:

- **Tool discovery**: The client asks the server what tools are available
- **Tool invocation**: The client calls a tool with parameters and gets results
- **Two transports**: `stdio` (subprocess) and `sse` (HTTP Server-Sent Events)

Learn more: [Model Context Protocol](https://modelcontextprotocol.io)

## How It Works in ClosedWheelerAGI

1. You configure MCP servers in your config file or via `/mcp add`
2. On startup (or `/mcp reload`), the agent connects to each enabled server
3. The agent discovers all tools from each server
4. Tools are registered with the prefix `mcp_<server>_<tool>` in the tool registry
5. The LLM can call these tools just like built-in tools

## Configuration

### Via Config File

Add `mcp_servers` to your `.agi/config.json`:

```json
{
  "mcp_servers": [
    {
      "name": "filesystem",
      "transport": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/dir"],
      "enabled": true
    },
    {
      "name": "github",
      "transport": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": ["GITHUB_TOKEN=ghp_your_token_here"],
      "enabled": true
    },
    {
      "name": "remote-api",
      "transport": "sse",
      "url": "http://localhost:8080/sse",
      "enabled": true
    }
  ]
}
```

### Via TUI Commands

```
/mcp add filesystem stdio npx -y @modelcontextprotocol/server-filesystem .
/mcp add my-api sse http://localhost:8080/sse
```

### Server Config Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique label for this server |
| `transport` | Yes | `"stdio"` or `"sse"` |
| `command` | stdio | Executable to run |
| `args` | No | Command-line arguments |
| `env` | No | Environment variables (e.g., `["KEY=value"]`) |
| `url` | sse | SSE endpoint URL |
| `enabled` | Yes | Set `false` to skip without removing |

## TUI Commands

| Command | Description |
|---------|-------------|
| `/mcp` or `/mcp list` | Show all configured servers and their status |
| `/mcp add <name> stdio <cmd> [args...]` | Add a stdio MCP server |
| `/mcp add <name> sse <url>` | Add an SSE MCP server |
| `/mcp remove <name>` | Disconnect and remove a server |
| `/mcp reload` | Disconnect all and reconnect (picks up config changes) |

## Transport Types

### stdio (Subprocess)

The agent launches the MCP server as a child process and communicates via stdin/stdout. This is the most common transport for local tools.

```json
{
  "name": "my-server",
  "transport": "stdio",
  "command": "node",
  "args": ["path/to/server.js"],
  "enabled": true
}
```

**Requirements:**
- The command must be in your PATH or use an absolute path
- The server must implement the MCP stdio transport protocol

### SSE (Server-Sent Events)

The agent connects to a remote MCP server over HTTP using Server-Sent Events. Use this for remote or shared servers.

```json
{
  "name": "remote",
  "transport": "sse",
  "url": "http://localhost:8080/sse",
  "enabled": true
}
```

## Popular MCP Servers

Here are some community MCP servers you can use:

| Server | Install Command | Description |
|--------|----------------|-------------|
| Filesystem | `npx -y @modelcontextprotocol/server-filesystem <path>` | Read/write files in a directory |
| GitHub | `npx -y @modelcontextprotocol/server-github` | GitHub API (issues, PRs, repos) |
| PostgreSQL | `npx -y @modelcontextprotocol/server-postgres <connstr>` | Query PostgreSQL databases |
| Brave Search | `npx -y @modelcontextprotocol/server-brave-search` | Web search via Brave |
| Memory | `npx -y @modelcontextprotocol/server-memory` | Persistent knowledge graph |
| Puppeteer | `npx -y @modelcontextprotocol/server-puppeteer` | Browser automation |
| SQLite | `npx -y @modelcontextprotocol/server-sqlite <path>` | SQLite database access |

For a full list, see: [MCP Servers Directory](https://github.com/modelcontextprotocol/servers)

## How Tools Are Named

MCP tools are registered with the naming pattern:

```
mcp_<server-name>_<tool-name>
```

For example, if you have a server named `filesystem` with a tool `read_file`, it becomes `mcp_filesystem_read_file` in the agent's tool registry.

The LLM sees these tools with their original descriptions prefixed by `[MCP:<server>]`.

## Examples

### Example 1: File System Server

```
/mcp add files stdio npx -y @modelcontextprotocol/server-filesystem C:\Projects
```

After connecting, tools like `mcp_files_read_file`, `mcp_files_write_file`, `mcp_files_list_directory` become available.

### Example 2: Custom Python MCP Server

Create a Python MCP server using the `mcp` package:

```python
# server.py
from mcp.server import Server
from mcp.server.stdio import stdio_server

app = Server("my-tools")

@app.tool()
async def calculate(expression: str) -> str:
    """Evaluate a math expression safely."""
    return str(eval(expression, {"__builtins__": {}}))

async def main():
    async with stdio_server() as (read, write):
        await app.run(read, write)

if __name__ == "__main__":
    import asyncio
    asyncio.run(main())
```

Then configure:
```json
{
  "name": "math",
  "transport": "stdio",
  "command": "python",
  "args": ["server.py"],
  "enabled": true
}
```

### Example 3: Remote SSE Server

If you have an MCP server running remotely:

```
/mcp add remote-tools sse https://mcp.example.com/sse
```

## Troubleshooting

**Server won't connect?**
- Check `.agi/debug.log` for `[MCP]` log entries
- Verify the command is in your PATH: try running it manually
- For stdio: ensure the server outputs valid MCP JSON-RPC on stdout
- For SSE: check the URL is accessible and the server is running

**Tools not appearing?**
- Run `/mcp list` to check connection status
- Run `/mcp reload` to force reconnection
- Verify the server's `ListTools` response is non-empty

**Tool calls failing?**
- Check that required parameters are provided
- Look at `.agi/debug.log` for detailed error messages
- The tool has a 60-second timeout per call

**Server disconnected?**
- Run `/mcp reload` to reconnect
- Check if the server process crashed (stdio) or the endpoint is down (SSE)

## Architecture

```
Agent
  |
  +-- Tool Registry (built-in tools + skills + MCP tools)
  |
  +-- MCP Manager
        |
        +-- Client: "filesystem" (stdio) --> npx mcp-server-filesystem
        |     Tools: mcp_filesystem_read_file, mcp_filesystem_write_file, ...
        |
        +-- Client: "github" (stdio) --> npx mcp-server-github
        |     Tools: mcp_github_create_issue, mcp_github_list_prs, ...
        |
        +-- Client: "remote" (sse) --> http://localhost:8080/sse
              Tools: mcp_remote_custom_tool, ...
```

The MCP Manager handles connection lifecycle, tool discovery, and bridges MCP tool calls into the agent's unified tool execution pipeline. All MCP tools go through the same security, permission, and retry infrastructure as built-in tools.
