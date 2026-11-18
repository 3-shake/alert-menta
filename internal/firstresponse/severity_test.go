package firstresponse

import "testing"

func TestDetermineSeverity(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		body     string
		expected SeverityLevel
	}{
		{
			name:     "High severity from label",
			labels:   []string{"severity:high"},
			body:     "",
			expected: SeverityHigh,
		},
		{
			name:     "High severity from sev1 label",
			labels:   []string{"sev1"},
			body:     "",
			expected: SeverityHigh,
		},
		{
			name:     "High severity from critical label",
			labels:   []string{"critical"},
			body:     "",
			expected: SeverityHigh,
		},
		{
			name:     "Medium severity from label",
			labels:   []string{"severity:medium"},
			body:     "",
			expected: SeverityMedium,
		},
		{
			name:     "Low severity from label",
			labels:   []string{"severity:low"},
			body:     "",
			expected: SeverityLow,
		},
		{
			name:     "Low severity from minor label",
			labels:   []string{"minor"},
			body:     "",
			expected: SeverityLow,
		},
		{
			name:     "High severity from body - production",
			labels:   []string{},
			body:     "Production server is down",
			expected: SeverityHigh,
		},
		{
			name:     "High severity from body - outage",
			labels:   []string{},
			body:     "Service outage affecting all users",
			expected: SeverityHigh,
		},
		{
			name:     "Low severity from body - staging",
			labels:   []string{},
			body:     "Issue in staging environment",
			expected: SeverityLow,
		},
		{
			name:     "Default to medium",
			labels:   []string{},
			body:     "Some generic issue",
			expected: SeverityMedium,
		},
		{
			name:     "Label takes precedence over body",
			labels:   []string{"severity:low"},
			body:     "Production server is down",
			expected: SeverityLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineSeverity(tt.labels, tt.body)
			if result != tt.expected {
				t.Errorf("DetermineSeverity(%v, %q) = %v, want %v",
					tt.labels, tt.body, result, tt.expected)
			}
		})
	}
}

func TestSeverityEmoji(t *testing.T) {
	tests := []struct {
		severity SeverityLevel
		expected string
	}{
		{SeverityHigh, "🔴"},
		{SeverityMedium, "🟡"},
		{SeverityLow, "🟢"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			result := SeverityEmoji(tt.severity)
			if result != tt.expected {
				t.Errorf("SeverityEmoji(%v) = %v, want %v",
					tt.severity, result, tt.expected)
			}
		})
	}
}
