# Skills System Guide

Skills are external scripts that extend ClosedWheelerAGI's tool capabilities. Each skill is a self-contained folder with a metadata file and a script that gets registered as a tool the LLM can call.

## How It Works

1. On startup, the agent scans `.agi/skills/` for skill folders
2. Each folder must contain a `skill.json` metadata file
3. The script referenced in `skill.json` is audited for security
4. If it passes, the skill is registered as a callable tool
5. The LLM can then invoke the skill just like any built-in tool

## Skills Directory

Skills are stored in the **application root**, not in the workspace:

```
<app-root>/
  .agi/
    skills/           <-- Skills live here
      my-skill/
        skill.json
        run.cmd
      another-skill/
        skill.json
        script.py
    config.json
    memory.json
```

The skills directory is automatically created when the agent starts.

## Creating a Skill

### 1. Create the skill folder

```
.agi/skills/hello-world/
```

### 2. Create `skill.json`

```json
{
  "name": "hello_world",
  "description": "Says hello to a person by name",
  "script": "run.cmd",
  "parameters": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "description": "The person's name"
      }
    },
    "required": ["name"]
  }
}
```

**Fields:**

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Tool name (used by the LLM to call it). Use `snake_case`. |
| `description` | Yes | What the tool does. The LLM reads this to decide when to use it. |
| `script` | Yes | Filename of the script to execute (relative to the skill folder). |
| `parameters` | No | JSON Schema describing the input parameters. |

### 3. Create the script

**Windows (`run.cmd`):**
```batch
@echo off
echo Hello, %1!
```

**Cross-platform (`run.py`):**
```python
import sys
name = sys.argv[1].replace("--name=", "") if len(sys.argv) > 1 else "World"
print(f"Hello, {name}!")
```

Parameters are passed as `--key=value` command-line arguments.

### 4. Reload skills

Use the TUI command:
```
/skill reload
```

Or restart the agent. Skills are also reloaded on `/reload`.

## TUI Commands

| Command | Description |
|---------|-------------|
| `/skill list` | Show all loaded skills and the skills directory path |
| `/skill reload` | Reload skills from disk (hot-reload without restart) |

## Examples

### File Counter Skill

```
.agi/skills/count-files/
  skill.json
  count.cmd
```

**skill.json:**
```json
{
  "name": "count_files",
  "description": "Counts files in a directory matching a pattern",
  "script": "count.cmd",
  "parameters": {
    "type": "object",
    "properties": {
      "directory": {
        "type": "string",
        "description": "Directory path to count files in"
      },
      "pattern": {
        "type": "string",
        "description": "File pattern (e.g., *.go, *.js)"
      }
    },
    "required": ["directory"]
  }
}
```

**count.cmd:**
```batch
@echo off
setlocal
set "dir=%~1"
set "pattern=%~2"
if "%dir%"=="" set "dir=."
if "%pattern%"=="" set "pattern=*.*"
set "dir=%dir:--directory=%"
set "pattern=%pattern:--pattern=%"
dir /b /s "%dir%\%pattern%" 2>nul | find /c /v ""
endlocal
```

### API Health Check Skill

```
.agi/skills/api-health/
  skill.json
  check.py
```

**skill.json:**
```json
{
  "name": "check_api_health",
  "description": "Checks if an API endpoint is responding",
  "script": "check.py",
  "parameters": {
    "type": "object",
    "properties": {
      "url": {
        "type": "string",
        "description": "The URL to check"
      }
    },
    "required": ["url"]
  }
}
```

## Security

- Every skill script is audited by the security system before registration
- Scripts that contain suspicious patterns (e.g., attempts to access sensitive system files) will be rejected
- Skills execute with a 30-second timeout
- Skills run within the application root context

## Troubleshooting

**Skill not loading?**
- Check `.agi/debug.log` for `[WARN] Failed to load skill` messages
- Verify `skill.json` has valid JSON and all required fields
- Ensure the script file exists and matches the `script` field name

**Skill not appearing in LLM tools?**
- Run `/skill list` to see if it's loaded
- Run `/skill reload` to force a reload
- Check that the skill name doesn't conflict with a built-in tool

**Script fails to execute?**
- Make sure the script is executable (on Linux/macOS: `chmod +x script.sh`)
- Check that the script interpreter is available (Python, Node.js, etc.)
- Test the script manually from the command line first
