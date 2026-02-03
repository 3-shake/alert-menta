# Go Development Rules

## Build Commands

```bash
# Using Makefile (recommended)
make help          # Show all available targets
make build         # Build all binaries with version info
make test          # Run tests with race detection and coverage
make lint          # Run golangci-lint
make ci            # Run lint, test, and build (for CI)
make dev-setup     # Set up development environment
make security      # Run gosec
make vuln          # Run govulncheck

# E2E tests (requires GITHUB_TOKEN and OPENAI_API_KEY)
make test-e2e
```

## Go Version & Tools

- Go 1.23 (specified in `go.mod`)
- golangci-lint v2.8.0 (golangci-lint-action v7)
- Install tools: `make tools`

## Code Quality Rules

- Run `make ci` before committing
- Run `make security` for security scans (gosec + govulncheck)
- Avoid `log.Fatal`/`log.Fatalf` in library code - return errors instead
- Check all errors from function calls (errcheck linter)
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Error strings should not be capitalized (staticcheck ST1005)

## Version Information

CLI binaries support `-version` flag:
```bash
./bin/alert-menta -version
# alert-menta v0.2.0
#   commit: abc1234
#   built:  2024-01-01T00:00:00Z
```

## Dependency Management

- Dependabot configured (`.github/dependabot.yml`)
- Weekly updates for Go modules and GitHub Actions
- Manual update: `make deps-update`

## E2E Testing

- Located in `e2e/` directory
- Requires `GITHUB_TOKEN` and `OPENAI_API_KEY`
- Run: `make test-e2e`
