//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func skipIfMissingEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN not set")
	}
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
}

func runCommand(t *testing.T, command string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "./cmd/main.go"}, args...)...)
	cmd.Dir = ".."
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestE2E_DescribeCommand(t *testing.T) {
	skipIfMissingEnv(t)

	output, err := runCommand(t,
		"go", "run", "./cmd/main.go",
		"-repo", "alert-menta",
		"-owner", "3-shake",
		"-issue", "1",
		"-github-token", os.Getenv("GITHUB_TOKEN"),
		"-api-key", os.Getenv("OPENAI_API_KEY"),
		"-command", "describe",
		"-config", ".alert-menta.user.yaml",
	)
	if err != nil {
		t.Fatalf("E2E describe command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Response:") {
		t.Errorf("Expected response output, got: %s", output)
	}
}

func TestE2E_SuggestCommand(t *testing.T) {
	skipIfMissingEnv(t)

	output, err := runCommand(t,
		"go", "run", "./cmd/main.go",
		"-repo", "alert-menta",
		"-owner", "3-shake",
		"-issue", "1",
		"-github-token", os.Getenv("GITHUB_TOKEN"),
		"-api-key", os.Getenv("OPENAI_API_KEY"),
		"-command", "suggest",
		"-config", ".alert-menta.user.yaml",
	)
	if err != nil {
		t.Fatalf("E2E suggest command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Response:") {
		t.Errorf("Expected response output, got: %s", output)
	}
}

func TestE2E_AskCommand(t *testing.T) {
	skipIfMissingEnv(t)

	output, err := runCommand(t,
		"go", "run", "./cmd/main.go",
		"-repo", "alert-menta",
		"-owner", "3-shake",
		"-issue", "1",
		"-github-token", os.Getenv("GITHUB_TOKEN"),
		"-api-key", os.Getenv("OPENAI_API_KEY"),
		"-command", "ask",
		"-intent", "What is the summary of this issue?",
		"-config", ".alert-menta.user.yaml",
	)
	if err != nil {
		t.Fatalf("E2E ask command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Response:") {
		t.Errorf("Expected response output, got: %s", output)
	}
}

func TestE2E_AnalysisCommand(t *testing.T) {
	skipIfMissingEnv(t)

	output, err := runCommand(t,
		"go", "run", "./cmd/main.go",
		"-repo", "alert-menta",
		"-owner", "3-shake",
		"-issue", "1",
		"-github-token", os.Getenv("GITHUB_TOKEN"),
		"-api-key", os.Getenv("OPENAI_API_KEY"),
		"-command", "analysis",
		"-config", ".alert-menta.user.yaml",
	)
	if err != nil {
		t.Fatalf("E2E analysis command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Response:") {
		t.Errorf("Expected response output, got: %s", output)
	}
}
