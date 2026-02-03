# alert-menta Makefile

.PHONY: all build test test-verbose test-e2e lint lint-fix fmt vet clean help deps deps-update tools dev-setup ci coverage release-dry-run security vuln

# Versions
GO_VERSION := 1.23
GOLANGCI_LINT_VERSION := v2.8.0

# Build
BUILD_DIR := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

all: lint test build

## Build
build:
	@echo "Building all binaries..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/alert-menta ./cmd/main.go
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/alert-menta-mcp ./cmd/mcp/main.go
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/alert-menta-firstresponse ./cmd/firstresponse/main.go
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/alert-menta-triage ./cmd/triage/main.go
	@echo "Build complete: $(BUILD_DIR)/"

## Build single binary (for backward compatibility)
build-main:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/alert-menta ./cmd/main.go

## Test
test:
	go test -race -cover -coverprofile=coverage.out ./...

test-verbose:
	go test -race -cover -v ./...

coverage: test
	go tool cover -html=coverage.out -o coverage.html

## E2E Tests (requires GITHUB_TOKEN and OPENAI_API_KEY)
test-e2e:
	go test -tags=e2e -v ./e2e/...

## Lint & Format
lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

fmt:
	go fmt ./...
	gofumpt -l -w .

vet:
	go vet ./...

## Dependencies
deps:
	go mod tidy
	go mod verify

deps-update:
	go get -u ./...
	go mod tidy

## Tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install mvdan.cc/gofumpt@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

## Clean
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## Development
dev-setup: tools deps
	@echo "Development environment ready"

## CI
ci: lint test build

## Release
release-dry-run:
	goreleaser release --snapshot --clean

## Security
security: vuln
	@echo "Running security checks..."
	gosec -quiet ./...

vuln:
	@echo "Checking for vulnerabilities..."
	govulncheck ./...

## Help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all             Run lint, test, and build"
	@echo "  build           Build all binaries with version info"
	@echo "  build-main      Build only the main CLI binary"
	@echo "  test            Run tests with race detection and coverage"
	@echo "  test-verbose    Run tests with verbose output"
	@echo "  test-e2e        Run E2E tests (requires GITHUB_TOKEN and OPENAI_API_KEY)"
	@echo "  coverage        Generate HTML coverage report"
	@echo "  lint            Run golangci-lint"
	@echo "  lint-fix        Run golangci-lint with auto-fix"
	@echo "  fmt             Format code with gofmt and gofumpt"
	@echo "  vet             Run go vet"
	@echo "  deps            Tidy and verify dependencies"
	@echo "  deps-update     Update dependencies"
	@echo "  tools           Install development tools"
	@echo "  security        Run security checks (gosec)"
	@echo "  vuln            Check for known vulnerabilities (govulncheck)"
	@echo "  clean           Remove build artifacts"
	@echo "  dev-setup       Set up development environment"
	@echo "  ci              Run CI checks (lint, test, build)"
	@echo "  release-dry-run Test release with goreleaser"
	@echo "  help            Show this help"
