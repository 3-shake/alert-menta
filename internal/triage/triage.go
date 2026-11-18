package triage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/3-shake/alert-menta/internal/ai"
)

// Triager performs AI-powered issue triage
type Triager struct {
	config   *Config
	aiClient ai.Ai
}

// NewTriager creates a new Triager
func NewTriager(config *Config, aiClient ai.Ai) *Triager {
	return &Triager{
		config:   config,
		aiClient: aiClient,
	}
}

// Triage analyzes an issue and returns triage results
func (t *Triager) Triage(title, body string, existingLabels []string) (*Result, error) {
	prompt := t.buildPrompt(title, body, existingLabels)

	response, err := t.aiClient.GetResponse(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI triage failed: %w", err)
	}

	return t.parseResponse(response)
}

// buildPrompt creates the triage prompt
func (t *Triager) buildPrompt(title, body string, existingLabels []string) *ai.Prompt {
	var sb strings.Builder

	sb.WriteString("You are an AI assistant that triages GitHub Issues.\n")
	sb.WriteString("Analyze the following Issue and determine the appropriate priority and category.\n\n")

	sb.WriteString("Available priority labels:\n")
	for _, label := range t.config.Labels.Priority {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", label.Name, label.Criteria))
	}

	sb.WriteString("\nAvailable category labels:\n")
	for _, label := range t.config.Labels.Category {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", label.Name, label.Criteria))
	}

	sb.WriteString("\nRespond with ONLY a valid JSON object in this exact format:\n")
	sb.WriteString(`{
  "priority": {"label": "priority:xxx", "confidence": 0.95},
  "category": {"label": "type:xxx", "confidence": 0.90},
  "reasoning": "Brief explanation of why these labels were chosen"
}`)
	sb.WriteString("\n\nDo not include any text before or after the JSON object.\n")

	userPrompt := fmt.Sprintf("Title: %s\n\nBody:\n%s", title, body)
	if len(existingLabels) > 0 {
		userPrompt += fmt.Sprintf("\n\nExisting labels: %s", strings.Join(existingLabels, ", "))
	}

	return &ai.Prompt{
		SystemPrompt: sb.String(),
		UserPrompt:   userPrompt,
		StructuredOutput: &ai.StructuredOutputOptions{
			Enabled:    true,
			SchemaName: "triage_result",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"priority": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"label":      map[string]interface{}{"type": "string"},
							"confidence": map[string]interface{}{"type": "number"},
						},
						"required": []string{"label", "confidence"},
					},
					"category": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"label":      map[string]interface{}{"type": "string"},
							"confidence": map[string]interface{}{"type": "number"},
						},
						"required": []string{"label", "confidence"},
					},
					"reasoning": map[string]interface{}{"type": "string"},
				},
				"required": []string{"priority", "category", "reasoning"},
			},
		},
	}
}

// parseResponse parses the AI response into a Result
func (t *Triager) parseResponse(response string) (*Result, error) {
	// Clean up response - remove markdown code blocks if present
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	var result Result
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse triage response: %w\nResponse: %s", err, response)
	}

	return &result, nil
}

// FormatComment formats the triage result as a GitHub comment
func (t *Triager) FormatComment(result *Result) string {
	var sb strings.Builder

	sb.WriteString("## 🤖 Auto-Triage Result\n\n")

	sb.WriteString("| Item | Label | Confidence |\n")
	sb.WriteString("|------|-------|------------|\n")
	sb.WriteString(fmt.Sprintf("| Priority | `%s` | %.0f%% |\n",
		result.Priority.Label, result.Priority.Confidence*100))
	sb.WriteString(fmt.Sprintf("| Category | `%s` | %.0f%% |\n",
		result.Category.Label, result.Category.Confidence*100))

	sb.WriteString("\n### Reasoning\n")
	sb.WriteString(result.Reasoning)
	sb.WriteString("\n\n")

	sb.WriteString("---\n")
	sb.WriteString("*This is an automated triage by AI. Please correct manually if needed.*\n")

	return sb.String()
}

// ShouldApplyLabel checks if a label should be applied based on confidence threshold
func (t *Triager) ShouldApplyLabel(confidence float64) bool {
	return confidence >= t.config.ConfidenceThreshold
}

// GetLabelsToApply returns the labels that should be applied
func (t *Triager) GetLabelsToApply(result *Result) []string {
	var labels []string

	if t.ShouldApplyLabel(result.Priority.Confidence) {
		labels = append(labels, result.Priority.Label)
	}

	if t.ShouldApplyLabel(result.Category.Confidence) {
		labels = append(labels, result.Category.Label)
	}

	return labels
}
