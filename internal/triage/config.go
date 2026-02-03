package triage

// Config holds the triage configuration
type Config struct {
	Enabled             bool              `yaml:"enabled" mapstructure:"enabled"`
	AutoLabel           bool              `yaml:"auto_label" mapstructure:"auto_label"`
	AutoComment         bool              `yaml:"auto_comment" mapstructure:"auto_comment"`
	ConfidenceThreshold float64           `yaml:"confidence_threshold" mapstructure:"confidence_threshold"`
	Labels              TriageLabelConfig `yaml:"labels" mapstructure:"labels"`
}

// TriageLabelConfig holds label configurations for triage
type TriageLabelConfig struct {
	Priority []LabelDefinition `yaml:"priority" mapstructure:"priority"`
	Category []LabelDefinition `yaml:"category" mapstructure:"category"`
}

// LabelDefinition defines a label and its criteria
type LabelDefinition struct {
	Name     string `yaml:"name" mapstructure:"name"`
	Criteria string `yaml:"criteria" mapstructure:"criteria"`
}

// Result holds the triage result
type Result struct {
	Priority  LabelResult `json:"priority"`
	Category  LabelResult `json:"category"`
	Reasoning string      `json:"reasoning"`
}

// LabelResult holds a single label result with confidence
type LabelResult struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

// DefaultConfig returns a default triage configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:             true,
		AutoLabel:           true,
		AutoComment:         true,
		ConfidenceThreshold: 0.7,
		Labels: TriageLabelConfig{
			Priority: []LabelDefinition{
				{Name: "priority:critical", Criteria: "Production service outage, data loss risk"},
				{Name: "priority:high", Criteria: "User impact, requires urgent attention"},
				{Name: "priority:medium", Criteria: "Feature degradation but workaround exists"},
				{Name: "priority:low", Criteria: "Improvement request, minor issue"},
			},
			Category: []LabelDefinition{
				{Name: "type:bug", Criteria: "Bug report for existing functionality"},
				{Name: "type:feature", Criteria: "New feature request"},
				{Name: "type:docs", Criteria: "Documentation update"},
				{Name: "type:incident", Criteria: "Incident report, alert"},
			},
		},
	}
}
