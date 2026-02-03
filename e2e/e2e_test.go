//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Test configuration - can be overridden via environment variables
func getTestRepo() string {
	if repo := os.Getenv("E2E_TEST_REPO"); repo != "" {
		return repo
	}
	return "alert-menta-test" // default test repo
}

func getTestOwner() string {
	if owner := os.Getenv("E2E_TEST_OWNER"); owner != "" {
		return owner
	}
	return "nwiizo" // default test owner
}

func getTestIssue() string {
	if issue := os.Getenv("E2E_TEST_ISSUE"); issue != "" {
		return issue
	}
	return "1" // default test issue
}

func skipIfMissingEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN not set")
	}
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
}

func getProjectRoot() string {
	// Get the directory of the test file, then go up one level
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filename))
}

func runCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	projectRoot := getProjectRoot()
	cmd := exec.Command("go", append([]string{"run", "./cmd/main.go"}, args...)...)
	cmd.Dir = projectRoot
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func getConfigPath() string {
	return filepath.Join(getProjectRoot(), ".alert-menta.user.yaml")
}

func TestE2E_DescribeCommand(t *testing.T) {
	skipIfMissingEnv(t)

	output, err := runCommand(t,
		"-repo", getTestRepo(),
		"-owner", getTestOwner(),
		"-issue", getTestIssue(),
		"-github-token", os.Getenv("GITHUB_TOKEN"),
		"-api-key", os.Getenv("OPENAI_API_KEY"),
		"-command", "describe",
		"-config", getConfigPath(),
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
		"-repo", getTestRepo(),
		"-owner", getTestOwner(),
		"-issue", getTestIssue(),
		"-github-token", os.Getenv("GITHUB_TOKEN"),
		"-api-key", os.Getenv("OPENAI_API_KEY"),
		"-command", "suggest",
		"-config", getConfigPath(),
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
		"-repo", getTestRepo(),
		"-owner", getTestOwner(),
		"-issue", getTestIssue(),
		"-github-token", os.Getenv("GITHUB_TOKEN"),
		"-api-key", os.Getenv("OPENAI_API_KEY"),
		"-command", "ask",
		"-intent", "What is the summary of this issue?",
		"-config", getConfigPath(),
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
		"-repo", getTestRepo(),
		"-owner", getTestOwner(),
		"-issue", getTestIssue(),
		"-github-token", os.Getenv("GITHUB_TOKEN"),
		"-api-key", os.Getenv("OPENAI_API_KEY"),
		"-command", "analysis",
		"-config", getConfigPath(),
	)
	if err != nil {
		t.Fatalf("E2E analysis command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Response:") {
		t.Errorf("Expected response output, got: %s", output)
	}
}

func TestE2E_PostmortemCommand(t *testing.T) {
	skipIfMissingEnv(t)

	output, err := runCommand(t,
		"-repo", getTestRepo(),
		"-owner", getTestOwner(),
		"-issue", getTestIssue(),
		"-github-token", os.Getenv("GITHUB_TOKEN"),
		"-api-key", os.Getenv("OPENAI_API_KEY"),
		"-command", "postmortem",
		"-config", getConfigPath(),
	)
	if err != nil {
		t.Fatalf("E2E postmortem command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Response:") {
		t.Errorf("Expected response output, got: %s", output)
	}
}
