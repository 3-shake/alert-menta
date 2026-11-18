# MCP Server Setup

alert-menta provides an MCP server for Claude Code integration.

## Running the Server

```bash
go run ./cmd/mcp/main.go -config .alert-menta.user.yaml
```

## Available Tools

| Tool | Description |
|------|-------------|
| `get_incident` | Get incident information from GitHub Issue |
| `analyze_incident` | Run analysis commands (describe, suggest, analysis, postmortem, runbook, timeline) |
| `post_comment` | Post a comment to GitHub Issue |
| `list_commands` | List all available commands |

## Claude Code Configuration

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "alert-menta": {
      "command": "go",
      "args": ["run", "./cmd/mcp/main.go", "-config", ".alert-menta.user.yaml"],
      "cwd": "/path/to/alert-menta",
      "env": {
        "GITHUB_TOKEN": "...",
        "OPENAI_API_KEY": "..."
      }
    }
  }
}
```
