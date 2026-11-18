package firstresponse

import "time"

// Config holds the first response guide configuration
type Config struct {
	Enabled       bool             `yaml:"enabled" mapstructure:"enabled"`
	TriggerLabels []string         `yaml:"trigger_labels" mapstructure:"trigger_labels"`
	Guides        []GuideConfig    `yaml:"guides" mapstructure:"guides"`
	DefaultGuide  string           `yaml:"default_guide" mapstructure:"default_guide"`
	Escalation    EscalationConfig `yaml:"escalation" mapstructure:"escalation"`
}

// GuideConfig holds configuration for a severity-specific guide
type GuideConfig struct {
	Severity   string   `yaml:"severity" mapstructure:"severity"`
	Template   string   `yaml:"template" mapstructure:"template"`
	AutoNotify []string `yaml:"auto_notify" mapstructure:"auto_notify"`
}

// EscalationConfig holds escalation settings
type EscalationConfig struct {
	Timeout      time.Duration `yaml:"timeout" mapstructure:"timeout"`
	NotifyTarget string        `yaml:"notify_target" mapstructure:"notify_target"`
}

// TemplateData holds data for rendering guide templates
type TemplateData struct {
	Issue             IssueSummary
	Severity          string
	SeverityEmoji     string
	OnCallTeam        []string
	SlackChannel      string
	EscalationTimeout string
	Commands          []CommandInfo
}

// IssueSummary contains issue information for templates
type IssueSummary struct {
	Number int
	Title  string
	URL    string
	Author string
	Labels []string
}

// CommandInfo describes an available command
type CommandInfo struct {
	Name        string
	Description string
}
