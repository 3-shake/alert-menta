# Troubleshooter Agent

このエージェントはClaude Codeが問題を自己診断・解決するための情報を提供します。

## GOROOT Misconfiguration

**症状**: ビルド時に `package flag is not in std (/Users/nwiizo/go/src/flag)` のようなエラー

**診断**:
```bash
go env GOROOT
# 正常: /opt/homebrew/Cellar/go/X.X.X/libexec
# 異常: /Users/nwiizo/go
```

**解決**:
1. シェル設定を確認: `~/.zshrc`, `~/.bashrc`
2. 誤ったGOROOT設定を削除
3. 正しい設定:
```bash
export GOROOT=$(brew --prefix go)/libexec
export PATH=$GOROOT/bin:$PATH
```

**ワークアラウンド**:
```bash
unset GOROOT && export GOROOT=/opt/homebrew/opt/go/libexec && go build ./...
```

## golangci-lint Version Mismatch

**症状**: CIとローカルでlint結果が異なる

**診断**: `golangci-lint --version` (期待値: v2.8.0)

**解決**: `make tools`

## golangci-lint v2 Config

**症状**: `unsupported version of the configuration` エラー

**解決**: `.golangci.yaml` 先頭に追加:
```yaml
version: "2"
```

**注意**: v2ではフォーマッタは `formatters:` セクションに記載

## E2E Test Failures

**症状**: E2Eテストがスキップされる

**診断**:
```bash
echo $GITHUB_TOKEN
echo $OPENAI_API_KEY
```

**解決**:
```bash
export GITHUB_TOKEN=your_token
export OPENAI_API_KEY=your_key
```

## CI E2E Tests Skipped

**症状**: CIでE2Eテストが全てSKIP

**原因**: リポジトリSecrets未設定

**解決**:
1. GitHub → Settings → Secrets → Actions
2. 追加: `GH_TOKEN`, `OPENAI_API_KEY`

## Viper Config Loading

**症状**: 新しい設定項目をviperが読み込まない

**原因**: `mapstructure` タグ不足

**解決**:
```go
type Config struct {
    WebhookURL string `yaml:"webhook_url" mapstructure:"webhook_url"`
}
```

## External Service Testing

**パターン**: httptest.NewServer でモック

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // 検証処理
    w.WriteHeader(http.StatusOK)
}))
defer server.Close()
client := NewClient(server.URL, "#channel")
```

## Cyclomatic Complexity

**症状**: `cyclomatic complexity X of func main is high`

**解決パターン**:
```go
func main() {
    cfg := parseFlags()
    if err := run(cfg); err != nil {
        log.Fatalf("Error: %v", err)
    }
}
func parseFlags() *Config { ... }
func run(cfg *Config) error { ... }
```

## Makefile Validation

```bash
make -n <target>  # ドライラン
make help         # 全ターゲット表示
```

バージョン埋め込み:
```makefile
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -X main.version=$(VERSION)
```
