# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

alert-menta is a GitHub Actions-based tool that uses LLM (OpenAI or VertexAI) to analyze and respond to GitHub Issues. It supports slash commands (`/describe`, `/suggest`, `/ask`) in Issue comments to generate AI-powered responses.

## Build and Development Commands

```bash
# Using Makefile (recommended)
make help          # Show all available targets
make build         # Build the binary
make test          # Run tests with race detection and coverage
make lint          # Run golangci-lint
make ci            # Run lint, test, and build (for CI)
make dev-setup     # Set up development environment

# E2E tests (requires GITHUB_TOKEN and OPENAI_API_KEY)
make test-e2e

# Manual commands
go build ./...
go test -race -cover ./...
golangci-lint run
go fmt ./...

# Local execution
go run ./cmd/main.go -repo <repository> -owner <owner> -issue <issue-number> \
  -github-token $GITHUB_TOKEN -api-key $OPENAI_API_KEY \
  -command <describe|suggest|ask> -config .alert-menta.user.yaml
```

## Architecture

```
cmd/main.go          # Entry point, CLI flag parsing, orchestrates the flow
internal/
  ai/
    ai.go            # Ai interface definition (GetResponse)
    openai.go        # OpenAI implementation (uses Azure SDK)
    vertexai.go      # VertexAI/Gemini implementation
  github/
    github.go        # GitHubIssue struct for Issue/comment operations
  slack/
    slack.go         # Slack webhook client for notifications
    slack_test.go    # Unit tests with mock HTTP server
  utils/
    utils.go         # Config loading (viper), image download/base64 conversion
```

**Flow**: Parse flags → Load config → Validate command → Fetch Issue+comments → Extract images → Build prompt → Call AI → Post response as comment

## Configuration

`.alert-menta.user.yaml` defines:
- `ai.provider`: "openai" or "vertexai"
- `ai.openai.model` / `ai.vertexai.{project,location,model}`
- `ai.commands`: Map of command names to `{description, system_prompt}`

## Branch Strategy

PRs should target the `develop` branch, not `main`. See wiki for details.

## Development Notes

### Go Version
- Project uses Go 1.23 (specified in `go.mod`)
- CI/CD workflows use Go 1.23
- golangci-lint v2.8.0 is used for linting (golangci-lint-action v7)

### Code Quality
- Run `make ci` before committing
- Avoid `log.Fatal`/`log.Fatalf` in library code - return errors instead
- Check all errors from function calls (errcheck linter)
- Use `fmt.Errorf("context: %w", err)` for error wrapping

### E2E Testing
- E2E tests are located in `e2e/` directory
- Tests require `GITHUB_TOKEN` and `OPENAI_API_KEY` environment variables
- Run with `make test-e2e` or `go test -tags=e2e -v ./e2e/...`

## Troubleshooting (for Claude Code)

このセクションはClaude Codeが問題を自己診断・解決するための情報です。

### GOROOT misconfiguration

**症状**: ビルド時に `package flag is not in std (/Users/nwiizo/go/src/flag)` のようなエラーが大量に出る

**診断コマンド**:
```bash
go env GOROOT
# 正常: /opt/homebrew/Cellar/go/X.X.X/libexec など
# 異常: /Users/nwiizo/go (ユーザーディレクトリ)
```

**原因**: GOROOTが誤ってユーザーディレクトリに設定されている

**解決方法**:
1. シェル設定ファイルを確認: `~/.zshrc`, `~/.bashrc`, `~/.zprofile`
2. 誤ったGOROOT設定を削除またはコメントアウト
3. 正しい設定に修正:
```bash
# ~/.zshrc に追記（必要な場合のみ）
export GOROOT=$(brew --prefix go)/libexec
export PATH=$GOROOT/bin:$PATH
```
4. シェルを再起動: `exec $SHELL`

**注意**: CIでは正常に動作するため、ローカル環境固有の問題

### golangci-lint version mismatch

**症状**: CIとローカルでlint結果が異なる

**診断コマンド**:
```bash
golangci-lint --version
# 期待値: v2.8.0
```

**解決方法**:
```bash
make tools
# または
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.8.0
```

### E2E test failures

**症状**: E2Eテストがスキップされる、または失敗する

**診断コマンド**:
```bash
echo $GITHUB_TOKEN
echo $OPENAI_API_KEY
```

**解決方法**: 環境変数が未設定の場合は設定する
```bash
export GITHUB_TOKEN=your_token
export OPENAI_API_KEY=your_key
```

### GOROOT問題のワークアラウンド

**症状**: シェル設定を変更できない/変更が反映されない場合

**ワークアラウンド**: コマンド実行時に一時的にGOROOTを設定
```bash
unset GOROOT && export GOROOT=/opt/homebrew/opt/go/libexec && go build ./...
unset GOROOT && export GOROOT=/opt/homebrew/opt/go/libexec && go test -race -cover ./...
unset GOROOT && export GOROOT=/opt/homebrew/opt/go/libexec && golangci-lint run
```

**注意**: このワークアラウンドは各コマンドに適用が必要。根本解決には環境変数の修正が必要。

### golangci-lint v2 設定形式

**症状**: `unsupported version of the configuration` エラー

**原因**: golangci-lint v2は設定ファイルに `version: "2"` が必要

**解決方法**: `.golangci.yaml` の先頭に追加
```yaml
version: "2"
```

**注意**: v2ではフォーマッタが `formatters:` セクションに移動。`gofmt`, `gofumpt`, `goimports` は `linters:` ではなく `formatters:` に記載。

### 設定構造体の追加パターン

**症状**: 新しい設定項目を追加してもviperが読み込まない

**原因**: `mapstructure` タグの不足、または構造体のフィールド名とYAMLキーの不一致

**解決方法**:
```go
// YAMLのsnake_caseキーには mapstructure タグが必要
type SlackConfig struct {
    Enabled    bool     `yaml:"enabled"`
    WebhookURL string   `yaml:"webhook_url" mapstructure:"webhook_url"`  // snake_case
    Channel    string   `yaml:"channel"`
    NotifyOn   []string `yaml:"notify_on" mapstructure:"notify_on"`      // snake_case
}
```

**注意**: viperはデフォルトで `mapstructure` を使用。YAMLキーがsnake_caseの場合、Goのフィールド名がCamelCaseなら `mapstructure` タグが必要。

### 外部サービス連携のテストパターン

**パターン**: Slack webhook等の外部サービス連携は httptest.NewServer でモック

```go
func TestSendCommandResponse(t *testing.T) {
    var receivedMessage Message

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // リクエストボディを検証
        if err := json.NewDecoder(r.Body).Decode(&receivedMessage); err != nil {
            t.Errorf("failed to decode: %v", err)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    client := NewClient(server.URL, "#test-channel")
    err := client.SendCommandResponse("Test", "http://test", "describe", "response")
    // ...
}
```

**利点**: 実際のAPIを呼び出さずにユニットテストが可能、CI/CDで安全に実行可能

### E2Eテストと環境変数

**設計**: E2Eテストは環境変数チェックでスキップ可能にする

```go
func skipIfMissingEnv(t *testing.T) {
    t.Helper()
    if os.Getenv("GITHUB_TOKEN") == "" {
        t.Skip("GITHUB_TOKEN not set")
    }
}
```

**理由**:
- ローカル開発では常に環境変数が設定されているとは限らない
- CIではGitHub Secretsから環境変数が供給される
- スキップにより `go test` が失敗しない
