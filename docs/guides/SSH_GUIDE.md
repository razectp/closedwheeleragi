# SSH Tools Guide

ClosedWheelerAGI includes built-in SSH tools that allow the agent to connect to remote servers, execute commands, and transfer files. SSH tools are **disabled by default** for security and must be explicitly enabled.

## Table of Contents

- [Enabling SSH Tools](#enabling-ssh-tools)
- [Connection Modes](#connection-modes)
  - [Visual Mode (Default)](#visual-mode-default)
  - [Hidden Mode](#hidden-mode)
- [Pre-Configured Hosts](#pre-configured-hosts)
- [Command Deny Lists](#command-deny-lists)
- [Available Tools](#available-tools)
  - [ssh_connect](#ssh_connect)
  - [ssh_exec](#ssh_exec)
  - [ssh_disconnect](#ssh_disconnect)
  - [ssh_list](#ssh_list)
  - [ssh_upload](#ssh_upload)
  - [ssh_download](#ssh_download)
- [Configuration Reference](#configuration-reference)
- [Usage Examples](#usage-examples)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)

---

## Enabling SSH Tools

SSH tools are disabled by default. You can enable them in two ways:

### Via Settings Overlay (TUI)

1. Press `F2` or use `/settings` to open the settings overlay
2. Navigate to **SSH Tools** and press `Enter` to toggle **ON**
3. Optionally toggle **SSH Visual Mode** (ON by default)
4. Restart the agent for changes to take effect

### Via config.json

Edit `.agi/config.json` in your project directory:

```json
{
  "ssh": {
    "enabled": true,
    "visual_mode": true
  }
}
```

After changing the config file, restart the agent.

---

## Connection Modes

### Visual Mode (Default)

When `visual_mode` is `true`, the agent connects **programmatically** (creating a managed SSH session) and **also** opens a monitor terminal window that tails a live log file. This lets the user watch every command the agent executes in real-time.

In visual mode:

- **Credentials come from config** — pre-configure hosts in `ssh.hosts` so the model never sees passwords or key paths
- `ssh_exec`, `ssh_upload`, and `ssh_download` all work normally
- A monitor window shows live command output via `tail -f` (Linux/macOS) or `Get-Content -Wait` (Windows)
- If the host is not pre-configured, the connection is rejected with a helpful error

This is the recommended mode because:

- The AI model has **no access** to authentication data
- You can see exactly what the agent does on your servers
- Full programmatic control is available (ssh_exec works)
- Per-host and global command deny lists protect against dangerous operations

### Hidden Mode

When `visual_mode` is `false`, the agent connects programmatically without opening a monitor window. The model can provide credentials directly via tool arguments, or use pre-configured hosts.

Hidden mode enables:

- `ssh_exec` — Run commands remotely and capture output
- `ssh_upload` — Upload files to the remote server
- `ssh_download` — Download files from the remote server

If no pre-configured host matches, the model must provide `user` and either `password` or `key_file`.

**Authentication methods (hidden mode without pre-configured host):**

| Method | Parameter | Notes |
|--------|-----------|-------|
| Password | `password` | Sent directly to the server |
| Key file | `key_file` | Path to a private key (e.g., `~/.ssh/id_rsa`) |

---

## Pre-Configured Hosts

Add hosts to `ssh.hosts` in your config so the agent can connect without ever seeing credentials:

```json
{
  "ssh": {
    "enabled": true,
    "visual_mode": true,
    "hosts": [
      {
        "label": "prod",
        "host": "192.168.1.100",
        "port": "22",
        "user": "deploy",
        "key_file": "~/.ssh/deploy_key"
      },
      {
        "label": "staging",
        "host": "10.0.0.5",
        "port": "2222",
        "user": "admin",
        "password": "s3cret"
      }
    ]
  }
}
```

The model connects by label: `ssh_connect {"host": "prod"}` — credentials are resolved from config.

In visual mode, **only pre-configured hosts are allowed**. In hidden mode, pre-configured hosts are used if available, otherwise the model provides credentials.

---

## Command Deny Lists

You can block dangerous commands globally or per-host using substring matching (case-insensitive):

```json
{
  "ssh": {
    "enabled": true,
    "deny_commands": [
      "rm -rf /",
      "mkfs",
      "dd if=",
      "shutdown",
      "reboot",
      "> /dev/sda"
    ],
    "hosts": [
      {
        "label": "prod",
        "host": "192.168.1.100",
        "user": "deploy",
        "key_file": "~/.ssh/deploy_key",
        "deny_commands": ["drop database", "truncate"]
      }
    ]
  }
}
```

- **Global deny patterns** (`ssh.deny_commands`) apply to all sessions
- **Per-host deny patterns** (`hosts[].deny_commands`) apply only to that host
- Both lists are checked: if any pattern matches (as a case-insensitive substring), the command is rejected
- Default global deny list includes: `rm -rf /`, `mkfs`, `dd if=`, `> /dev/sda`, `shutdown`, `reboot`, `init 0`, `halt`

---

## Available Tools

### ssh_connect

Establishes an SSH connection to a remote server.

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `host` | Yes | Remote host address, IP, or label of a pre-configured host |
| `port` | No | SSH port (default: `22`, ignored if host config has port) |
| `label` | No | Session label for referencing later (default: same as host) |
| `user` | Hidden mode only | SSH username (not needed for pre-configured hosts) |
| `password` | Hidden mode only | SSH password (not needed for pre-configured hosts) |
| `key_file` | Hidden mode only | Path to SSH private key (not needed for pre-configured hosts) |

**Visual mode behavior:**
- Looks up host in `ssh.hosts` config by label or hostname
- Connects programmatically using config credentials
- Opens a monitor terminal window tailing `.agi/ssh_<label>.log`
- Returns error if host is not pre-configured

**Hidden mode behavior:**
- Looks up host in `ssh.hosts` first; if not found, uses provided credentials
- Connects programmatically
- No monitor window

**Example (pre-configured host):**
```
Tool: ssh_connect
Args: {"host": "prod"}
```

**Example (hidden mode, ad-hoc):**
```
Tool: ssh_connect
Args: {"host": "192.168.1.100", "user": "deploy", "key_file": "/home/user/.ssh/id_rsa", "label": "prod"}
```

---

### ssh_exec

Executes a command on an active SSH session. Works in **both** visual and hidden mode.

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `label` | Yes | Session label from `ssh_connect` |
| `command` | Yes | Shell command to execute |
| `timeout` | No | Timeout in seconds (default: `30`) |

Commands are checked against global and per-host deny lists before execution. Returns both `stdout` and `stderr`. In visual mode, output is also written to the monitor log file.

**Example:**
```
Tool: ssh_exec
Args: {"label": "prod", "command": "df -h", "timeout": "10"}
```

---

### ssh_disconnect

Closes an active SSH session and cleans up the log file.

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `label` | Yes | Session label to disconnect |

**Example:**
```
Tool: ssh_disconnect
Args: {"label": "prod"}
```

---

### ssh_list

Lists all active SSH sessions with their labels, user, host, mode (visual/hidden), and connection time.

**Parameters:** None.

**Example output:**
```
Active SSH sessions (2):
  - prod (deploy@192.168.1.100:22, visual, since 14:32:10)
  - staging (admin@10.0.0.5:2222, hidden, since 14:35:22)
```

---

### ssh_upload

Uploads a local file to a remote server over an active SSH session. Uses `cat >` via stdin pipe.

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `label` | Yes | Session label |
| `local_path` | Yes | Path to the local file |
| `remote_path` | Yes | Destination path on the remote server |

**Example:**
```
Tool: ssh_upload
Args: {"label": "prod", "local_path": "workplace/deploy.sh", "remote_path": "/tmp/deploy.sh"}
```

---

### ssh_download

Downloads a file from a remote server to the local filesystem. Uses `cat` to read the remote file.

**Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `label` | Yes | Session label |
| `remote_path` | Yes | Path to the file on the remote server |
| `local_path` | Yes | Local destination path |

Parent directories are created automatically if they don't exist.

**Example:**
```
Tool: ssh_download
Args: {"label": "prod", "remote_path": "/var/log/app.log", "local_path": "workplace/logs/app.log"}
```

---

## Configuration Reference

All SSH settings live under the `ssh` key in `.agi/config.json`:

```json
{
  "ssh": {
    "enabled": true,
    "visual_mode": true,
    "deny_commands": [
      "rm -rf /",
      "mkfs",
      "dd if=",
      "> /dev/sda",
      "shutdown",
      "reboot",
      "halt",
      "init 0"
    ],
    "hosts": [
      {
        "label": "prod",
        "host": "192.168.1.100",
        "port": "22",
        "user": "deploy",
        "key_file": "~/.ssh/deploy_key",
        "deny_commands": ["drop database", "truncate"]
      },
      {
        "label": "staging",
        "host": "10.0.0.5",
        "port": "2222",
        "user": "admin",
        "password": "s3cret"
      },
      {
        "label": "dev",
        "host": "localhost",
        "user": "dev",
        "key_file": "~/.ssh/id_rsa"
      }
    ]
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ssh.enabled` | `bool` | `false` | Enable/disable SSH tools (restart required) |
| `ssh.visual_mode` | `bool` | `true` | Open monitor window + use config credentials |
| `ssh.deny_commands` | `[]string` | *(see above)* | Global command deny patterns (substring match) |
| `ssh.hosts` | `[]object` | `[]` | Pre-configured host entries |
| `ssh.hosts[].label` | `string` | required | Unique label to reference this host |
| `ssh.hosts[].host` | `string` | required | Hostname or IP address |
| `ssh.hosts[].port` | `string` | `"22"` | SSH port |
| `ssh.hosts[].user` | `string` | | SSH username |
| `ssh.hosts[].password` | `string` | | SSH password |
| `ssh.hosts[].key_file` | `string` | | Path to private key file |
| `ssh.hosts[].deny_commands` | `[]string` | `[]` | Per-host command deny patterns |

---

## Usage Examples

### Example 1: Visual Mode — Agent deploys while user watches

1. Configure hosts and enable SSH with visual mode in config
2. Ask the agent: *"Connect to prod and check disk usage"*
3. The agent calls `ssh_connect {"host": "prod"}`
4. A monitor window opens showing live command output
5. The agent runs `ssh_exec {"label": "prod", "command": "df -h"}`
6. You see the command and output in the monitor window in real-time

### Example 2: Hidden Mode — Agent manages a deployment

1. Set `visual_mode` to `false` in settings
2. Ask the agent: *"Connect to the staging server and deploy the latest build"*
3. The agent calls `ssh_connect` with credentials (from config or args), then:
   - `ssh_exec` to pull latest code
   - `ssh_exec` to restart services
   - `ssh_exec` to verify deployment health

**Agent workflow:**
```
1. ssh_connect  {"host": "staging"}
2. ssh_exec     {"label": "staging", "command": "cd /app && git pull origin main"}
3. ssh_exec     {"label": "staging", "command": "systemctl restart myapp"}
4. ssh_exec     {"label": "staging", "command": "curl -s http://localhost:8080/health"}
5. ssh_disconnect {"label": "staging"}
```

### Example 3: File Transfer

```
1. ssh_connect  {"host": "prod"}
2. ssh_download {"label": "prod", "remote_path": "/var/backups/db.sql.gz", "local_path": "workplace/db_backup.sql.gz"}
3. ssh_upload   {"label": "prod", "local_path": "workplace/restore.sh", "remote_path": "/tmp/restore.sh"}
4. ssh_exec     {"label": "prod", "command": "chmod +x /tmp/restore.sh && /tmp/restore.sh"}
5. ssh_disconnect {"label": "prod"}
```

### Example 4: Deny command in action

With `deny_commands: ["drop database"]` on the prod host:

```
Tool: ssh_exec
Args: {"label": "prod", "command": "mysql -e 'DROP DATABASE users'"}
Result: Error — command denied by policy: matches host pattern "drop database"
```

---

## Security Considerations

### Credential Handling

- **Visual mode (recommended):** Credentials live in `config.json` only. The AI model references hosts by label and never sees passwords or key paths.
- **Hidden mode:** If the host is not pre-configured, credentials are passed as tool arguments and the model sees them. Use pre-configured hosts whenever possible.

### Host Key Verification

Host key verification is **disabled** (`InsecureIgnoreHostKey`) in both modes. This is intentional for the agent use case but means connections are vulnerable to man-in-the-middle attacks on untrusted networks.

### Command Deny Lists

The default global deny list blocks common destructive commands:
- `rm -rf /`, `mkfs`, `dd if=`, `> /dev/sda`, `shutdown`, `reboot`, `init 0`, `halt`

Add per-host deny patterns for database or application-specific dangerous operations (e.g., `drop database`, `truncate`).

### Sensitive Tool Classification

The following SSH tools are classified as **sensitive** in the security system:
- `ssh_connect` — Establishes connections
- `ssh_exec` — Executes remote commands
- `ssh_upload` — Writes files to remote servers

Sensitive tool executions are logged to `.agi/audit.log`.

### Session Cleanup

All active SSH sessions are automatically closed when:
- The agent shuts down normally
- The TUI is closed
- The agent is restarted

Log files (`ssh_<label>.log`) and launcher scripts are stored in `.agi/`.

### Recommendations

1. **Use visual mode** with pre-configured hosts for maximum security
2. **Configure deny lists** for production hosts to prevent destructive commands
3. **Use key-based auth** over passwords whenever possible
4. **Review audit.log** periodically to see SSH tool usage
5. **Disable SSH tools** when not needed — keep them off by default
6. **Never commit** SSH credentials or key files to version control

---

## Troubleshooting

### SSH tools not appearing

- Ensure `ssh.enabled` is `true` in `.agi/config.json`
- **Restart the agent** after changing the setting (SSH tools are registered at startup)

### Visual mode: "host is not pre-configured"

- Add the host to `ssh.hosts` in config.json with label, host, user, and password/key_file
- Or switch to hidden mode (`ssh.visual_mode: false`) to provide credentials via args

### Visual mode: monitor window doesn't open (Linux)

Install one of: `gnome-terminal`, `konsole`, `xfce4-terminal`, `alacritty`, `kitty`, `wezterm`, `foot`, `tilix`, or `xterm`.

```bash
# Ubuntu/Debian
sudo apt install gnome-terminal

# Fedora
sudo dnf install gnome-terminal

# Arch
sudo pacman -S gnome-terminal
```

### Visual mode: monitor window doesn't open (Windows)

- Windows Terminal (`wt.exe`) is preferred but `cmd.exe` works as fallback
- Check `.agi/ssh_monitor_<label>.cmd` for the generated launcher script

### Hidden mode: "connection refused"

- Verify the host is reachable: `ping <host>`
- Check the SSH port is open: `Test-NetConnection <host> -Port 22` (PowerShell)
- Verify credentials are correct

### Hidden mode: "failed to parse key file"

- Ensure the key file exists and is readable
- The key must be in PEM format (OpenSSH format keys may need conversion)
- Passphrase-protected keys are **not supported** — use an unprotected key or add the host to config

### ssh_exec: "command denied by policy"

- The command matches a pattern in `ssh.deny_commands` (global) or `hosts[].deny_commands` (per-host)
- Review deny patterns in config; matching is case-insensitive substring
- Remove the pattern if the command should be allowed

### ssh_exec: "command timed out"

- Increase the `timeout` parameter (default: 30 seconds)
- For long-running commands, consider running them in the background: `nohup <command> &`
- Partial output before the timeout is returned in the result

### ssh_upload/download: large files

- The upload/download tools read the entire file into memory
- For very large files (>100MB), consider using `scp` or `rsync` via ssh_exec instead
- Binary files are supported but very large binaries may cause memory pressure
