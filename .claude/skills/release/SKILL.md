# Release Workflow

## Binaries Built

| Binary | Description |
|--------|-------------|
| `alert-menta` | Main CLI |
| `alert-menta-mcp` | MCP server for Claude Code |
| `alert-menta-firstresponse` | First response guide generator |
| `alert-menta-triage` | Auto-triage for issues |

## Supported Platforms

- linux/amd64, linux/arm64
- darwin/amd64, darwin/arm64
- windows/amd64, windows/arm64

## Release Commands

```bash
# Test release locally (no publish)
make release-dry-run

# Actual release (triggered by tag push)
git tag v0.2.0
git push origin v0.2.0
```

## Configuration Files

- GoReleaser: `.goreleaser.yaml`
- Workflow: `.github/workflows/release.yaml`
