package utils

import (
	"os"
	"testing"
)

// TestNewConfig tests the NewConfig function
func TestNewConfig(t *testing.T) {
	// Setup: Create a temporary config file
	configContent := `
system:
  debug:
    log_level: "debug"
ai:
  provider: "openai"
  openai:
    model: "text-davinci-003"
  commands:
    command1:
      description: "Test command"
      system_prompt: "Prompt"
`
	tempFile, err := os.CreateTemp("", "testconfig*.yaml")
	if err != nil {
		t.Fatalf("Error creating temporary config file: %v", err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }() // Clean up after the test

	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Error writing to temporary config file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Error closing temporary config file: %v", err)
	}

	// Test: Call NewConfig
	cfg, err := NewConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("NewConfig returned an error: %v", err)
	}

	// Validate: Check if the values are correctly parsed
	if cfg.System.Debug.LogLevel != "debug" {
		t.Errorf("Expected log_level 'debug', got '%s'", cfg.System.Debug.LogLevel)
	}
	if cfg.Ai.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", cfg.Ai.Provider)
	}
	if cfg.Ai.OpenAI.Model != "text-davinci-003" {
		t.Errorf("Expected model 'text-davinci-003', got '%s'", cfg.Ai.OpenAI.Model)
	}
	if cfg.Ai.Commands["command1"].Description != "Test command" {
		t.Errorf("Expected command description 'Test command', got '%s'", cfg.Ai.Commands["command1"].Description)
	}
	if cfg.Ai.Commands["command1"].SystemPrompt != "Prompt" {
		t.Errorf("Expected system_prompt 'Prompt', got '%s'", cfg.Ai.Commands["command1"].SystemPrompt)
	}
}
