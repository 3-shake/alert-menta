# CLAUDE.md

## Project Overview

alert-menta is a GitHub Actions-based tool that uses LLM (OpenAI, Anthropic, VertexAI) to analyze and respond to GitHub Issues. Supports slash commands (`/describe`, `/suggest`, `/ask`, `/analysis`, `/postmortem`, `/runbook`, `/timeline`) in Issue comments.

## Quick Commands

```bash
make ci            # lint, test, build (before commit)
make test-e2e      # E2E tests (requires GITHUB_TOKEN, OPENAI_API_KEY)
make release-dry-run  # Test GoReleaser
```

## Architecture

```
cmd/
  main.go              # Main CLI
  mcp/main.go          # MCP server for Claude Code
  triage/main.go       # Auto-triage CLI
  firstresponse/main.go # First response guide CLI
internal/
  ai/                  # AI providers (OpenAI, Anthropic, VertexAI)
  github/              # GitHub API wrapper
  mcp/                 # MCP server implementation
  triage/              # Auto-triage logic
  firstresponse/       # First response logic
  slack/               # Slack notifications
  utils/               # Config, image processing
```

**Flow**: Parse flags → Load config → Fetch Issue → Build prompt → Call AI → Post comment

## Configuration

`.alert-menta.user.yaml`: AI provider, models, commands, Slack notifications

## Documentation

| Topic | Location |
|-------|----------|
| Go development rules | [.claude/rules/go-development.md](.claude/rules/go-development.md) |
| Git workflow | [.claude/rules/git-workflow.md](.claude/rules/git-workflow.md) |
| MCP server setup | [.claude/skills/mcp-server/SKILL.md](.claude/skills/mcp-server/SKILL.md) |
| Release workflow | [.claude/skills/release/SKILL.md](.claude/skills/release/SKILL.md) |
| Troubleshooting | [.claude/agents/troubleshooter.md](.claude/agents/troubleshooter.md) |
