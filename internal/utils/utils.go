package utils

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/spf13/viper"
)

// Root structure of information read from config file
type Config struct {
	System        System              `yaml:"system"`
	Ai            Ai                  `yaml:"ai"`
	Notifications Notifications       `yaml:"notifications"`
	FirstResponse FirstResponseConfig `yaml:"first_response" mapstructure:"first_response"`
}

// FirstResponseConfig holds first response guide configuration
type FirstResponseConfig struct {
	Enabled       bool                      `yaml:"enabled" mapstructure:"enabled"`
	TriggerLabels []string                  `yaml:"trigger_labels" mapstructure:"trigger_labels"`
	Guides        []FirstResponseGuide      `yaml:"guides" mapstructure:"guides"`
	DefaultGuide  string                    `yaml:"default_guide" mapstructure:"default_guide"`
	SlackChannel  string                    `yaml:"slack_channel" mapstructure:"slack_channel"`
	Escalation    FirstResponseEscalation   `yaml:"escalation" mapstructure:"escalation"`
}

// FirstResponseGuide holds configuration for severity-specific guides
type FirstResponseGuide struct {
	Severity   string   `yaml:"severity" mapstructure:"severity"`
	Template   string   `yaml:"template" mapstructure:"template"`
	AutoNotify []string `yaml:"auto_notify" mapstructure:"auto_notify"`
}

// FirstResponseEscalation holds escalation settings
type FirstResponseEscalation struct {
	TimeoutMinutes int    `yaml:"timeout_minutes" mapstructure:"timeout_minutes"`
	NotifyTarget   string `yaml:"notify_target" mapstructure:"notify_target"`
}

// Notifications holds notification configuration
type Notifications struct {
	Slack SlackConfig `yaml:"slack"`
}

// SlackConfig holds Slack notification settings
type SlackConfig struct {
	Enabled    bool     `yaml:"enabled"`
	WebhookURL string   `yaml:"webhook_url" mapstructure:"webhook_url"`
	Channel    string   `yaml:"channel"`
	NotifyOn   []string `yaml:"notify_on" mapstructure:"notify_on"`
}

type System struct {
	Debug SystemDebug `yaml:"debug"`
}

type SystemDebug struct {
	LogLevel string `yaml:"log_level" mapstructure:"log_level"`
}

type Ai struct {
	Commands  map[string]Command `yaml:"commands"`
	Provider  string             `yaml:"provider"`
	OpenAI    OpenAI             `yaml:"openai"`
	VertexAI  VertexAI           `yaml:"vertexai"`
	Anthropic AnthropicConfig    `yaml:"anthropic"`
	Fallback  FallbackConfig     `yaml:"fallback"`
}

// FallbackConfig holds fallback provider configuration
type FallbackConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Providers []string `yaml:"providers" mapstructure:"providers"`
	Retry     RetryConfig `yaml:"retry"`
}

// RetryConfig holds retry settings for fallback
type RetryConfig struct {
	MaxRetries int `yaml:"max_retries" mapstructure:"max_retries"`
	DelayMs    int `yaml:"delay_ms" mapstructure:"delay_ms"`
	TimeoutMs  int `yaml:"timeout_ms" mapstructure:"timeout_ms"`
}

type Command struct {
	Description      string                  `yaml:"description"`
	SystemPrompt     string                  `yaml:"system_prompt" mapstructure:"system_prompt"`
	RequireIntent    bool                    `yaml:"require_intent" mapstructure:"require_intent"`
	StructuredOutput *StructuredOutputConfig `yaml:"structured_output,omitempty" mapstructure:"structured_output"`
}

// StructuredOutputConfig holds structured output settings for a command
type StructuredOutputConfig struct {
	Enabled        bool                   `yaml:"enabled" mapstructure:"enabled"`
	Schema         map[string]interface{} `yaml:"schema,omitempty" mapstructure:"schema"`
	SchemaName     string                 `yaml:"schema_name,omitempty" mapstructure:"schema_name"`
	FallbackToText bool                   `yaml:"fallback_to_text" mapstructure:"fallback_to_text"`
}

type OpenAI struct {
	Model string `yaml:"model"`
}

type VertexAI struct {
	Model   string `yaml:"model"`
	Project string `yaml:"project"`
	Region  string `yaml:"region"`
}

type AnthropicConfig struct {
	Model string `yaml:"model"`
}

func NewConfig(filename string) (*Config, error) {
	// Initialize a logger
	logger := log.New(
		os.Stdout, "[alert-menta utils] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	// Get the directory and file name from variable filename
	dir, file := filepath.Split(filename)
	// Extract base part and extension part
	base, ext := filepath.Base(file)[:len(filepath.Base(file))-len(filepath.Ext(file))], filepath.Ext(file)[1:]

	// Read the config file
	viper.SetConfigName(base)
	viper.SetConfigType(ext)
	viper.AddConfigPath(dir)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Unmarshal the config file
	cfg := new(Config)
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Print the config
	logger.Println("Config:", cfg)
	return cfg, nil
}

func DownloadImage(url string, token string) ([]byte, string, error) {
	// Create a new HTTP client
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to create a new request: %w", err)
	}

	// Download the image with the token
	req.Header.Set("Authorization", "Bearer "+token) // set token to header
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to get a response: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Write the response body to the temporary file
	file, err := os.CreateTemp("", "downloaded-image-*")
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to create a temporary file: %w", err)
	}
	defer func() {
		log.Println("remove", file.Name(), "Content-Type:", resp.Header.Get("Content-Type"))
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to write the response body to the temporary file: %w", err)
	}

	// Read image data from the temporary file
	data, err := os.ReadFile(file.Name())
	if err != nil {
		return []byte{}, "", fmt.Errorf("failed to read the file: %w", err)
	}

	// Get the extension of the image
	contentType := resp.Header.Get("Content-Type")
	imageRegex := regexp.MustCompile(`.+/(.*)`)
	matches := imageRegex.FindAllStringSubmatch(contentType, -1)
	if len(matches) == 0 {
		return []byte{}, "", fmt.Errorf("failed to get the extension of the image")
	}
	ext := matches[0][1]

	return data, ext, nil
}

func ImageToBase64(data []byte, ext string) string {
	base64img := base64.StdEncoding.EncodeToString(data)
	return "data:image/" + ext + ";base64," + base64img
}

func ExtractImageURLs(body string) []string {
	imageRegex := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
	matches := imageRegex.FindAllStringSubmatch(body, -1)
	var urls []string
	for _, match := range matches {
		urls = append(urls, match[2])
	}
	return urls
}
