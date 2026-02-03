package main

import (
	"errors"
	"log"
	"os"
	"testing"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/utils"
)

// Test for validateCommand
func TestValidateCommand(t *testing.T) {
	mockCfg := &utils.Config{
		Ai: utils.Ai{
			Commands: map[string]utils.Command{
				"valid": {SystemPrompt: "hoge"},
			},
		},
	}

	tests := []struct {
		command  string
		expected error
	}{
		{"valid", nil},
		{"invalid", errors.New("invalid command: invalid, allowed commands are valid")},
	}

	for _, tt := range tests {
		err := validateCommand(tt.command, mockCfg)
		if err != nil && err.Error() != tt.expected.Error() {
			t.Errorf("expected %v, got %v", tt.expected, err)
		}
	}
}

// Test for constructPrompt
func TestConstructPrompt(t *testing.T) {
	mockCfg := &utils.Config{
		Ai: utils.Ai{
			Commands: map[string]utils.Command{
				"ask":   {SystemPrompt: "Ask system prompt: ", RequireIntent: true},
				"other": {SystemPrompt: "Other system prompt: ", RequireIntent: false},
			},
		},
	}

	// Logger setup for testing
	logger := log.New(os.Stdout, "", 0)

	tests := []struct {
		name                 string
		command              string
		intent               string
		userPrompt           string
		imgs                 []ai.Image
		expectErr            bool
		expectedSystemPrompt string
	}{
		{"Valid Ask Command with Intent", "ask", "What is the first thing to work on in suggestions?", "userPrompt", []ai.Image{}, false, "Ask system prompt: What is the first thing to work on in suggestions?\n"},
		{"Ask Command without Intent", "ask", "", "userPrompt", []ai.Image{}, true, ""},
		{"Valid Other Command", "other", "", "userPrompt", []ai.Image{}, false, "Other system prompt: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := constructPrompt(tt.command, tt.intent, tt.userPrompt, tt.imgs, mockCfg, logger)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got error %v", tt.expectErr, err)
			}
			if err == nil {
				if prompt.SystemPrompt != tt.expectedSystemPrompt {
					t.Errorf("expected system prompt: %s, got %s", tt.expectedSystemPrompt, prompt.SystemPrompt)
				}
				if prompt.UserPrompt != tt.userPrompt {
					t.Errorf("expected user prompt: %s, got %s", tt.userPrompt, prompt.UserPrompt)
				}
			}
		})
	}
}

// Test for getAIClient
func TestGetAIClient(t *testing.T) {
	mockCfg := &utils.Config{
		Ai: utils.Ai{
			Provider: "openai",
			OpenAI: utils.OpenAI{
				Model: "test-model",
			},
		},
	}

	tests := []struct {
		oaiKey    string
		expectErr bool
		provider  string
	}{
		{"valid-key", false, "openai"},
		{"", true, "openai"},
		{"", true, "invalid"},
	}

	for _, tt := range tests {
		mockCfg.Ai.Provider = tt.provider
		_, err := getAIClient(tt.oaiKey, mockCfg, log.New(os.Stdout, "", 0))
		if (err != nil) != tt.expectErr {
			t.Errorf("expected error: %v, got %v", tt.expectErr, err)
		}
	}
}

// Test for getAvailableCommands
func TestGetAvailableCommands(t *testing.T) {
	mockCfg := &utils.Config{
		Ai: utils.Ai{
			Commands: map[string]utils.Command{
				"valid": {Description: "Valid command"},
				"other": {Description: "Other command"},
			},
		},
	}
	commands := getAvailableCommands(mockCfg)
	if len(commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(commands))
	}
}

// Test for commandNeedsIntent
func TestCommandNeedsIntent(t *testing.T) {
	mockCfg := &utils.Config{
		Ai: utils.Ai{
			Commands: map[string]utils.Command{
				"ask":   {SystemPrompt: "Ask system prompt: ", RequireIntent: true},
				"other": {SystemPrompt: "Other system prompt: ", RequireIntent: false},
			},
		},
	}

	tests := []struct {
		command     string
		shouldNeed  bool
		expectError bool
	}{
		{"ask", true, false},
		{"other", false, false},
		{"nonexistent", false, true},
	}

	for _, tt := range tests {
		needsIntent, err := commandNeedsIntent(tt.command, mockCfg)

		// Check if error condition matches expectation
		if (err != nil) != tt.expectError {
			t.Errorf("expected error: %v, got: %v for command %s", tt.expectError, err != nil, tt.command)
		}

		// If not expecting an error, check the result
		if !tt.expectError && needsIntent != tt.shouldNeed {
			t.Errorf("expected %v for command %s, got %v", tt.shouldNeed, tt.command, needsIntent)
		}
	}
}
