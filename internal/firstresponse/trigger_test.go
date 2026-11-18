package firstresponse

import "testing"

func TestShouldTrigger(t *testing.T) {
	tests := []struct {
		name          string
		issueLabels   []string
		triggerLabels []string
		expected      bool
	}{
		{
			name:          "Match single label",
			issueLabels:   []string{"incident", "bug"},
			triggerLabels: []string{"incident"},
			expected:      true,
		},
		{
			name:          "Match one of multiple trigger labels",
			issueLabels:   []string{"alert"},
			triggerLabels: []string{"incident", "alert", "outage"},
			expected:      true,
		},
		{
			name:          "No match",
			issueLabels:   []string{"bug", "enhancement"},
			triggerLabels: []string{"incident", "alert"},
			expected:      false,
		},
		{
			name:          "Case insensitive match",
			issueLabels:   []string{"INCIDENT"},
			triggerLabels: []string{"incident"},
			expected:      true,
		},
		{
			name:          "Empty trigger labels",
			issueLabels:   []string{"incident"},
			triggerLabels: []string{},
			expected:      false,
		},
		{
			name:          "Empty issue labels",
			issueLabels:   []string{},
			triggerLabels: []string{"incident"},
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldTrigger(tt.issueLabels, tt.triggerLabels)
			if result != tt.expected {
				t.Errorf("ShouldTrigger(%v, %v) = %v, want %v",
					tt.issueLabels, tt.triggerLabels, result, tt.expected)
			}
		})
	}
}

func TestHasExistingGuide(t *testing.T) {
	tests := []struct {
		name     string
		comments []string
		expected bool
	}{
		{
			name:     "Has guide marker",
			comments: []string{"Some comment", "## 🚨 インシデント対応ガイド\nMore content"},
			expected: true,
		},
		{
			name:     "Has English guide marker",
			comments: []string{"## Incident Response Guide"},
			expected: true,
		},
		{
			name:     "Has HTML marker",
			comments: []string{"<!-- alert-menta:first-response -->\nGuide content"},
			expected: true,
		},
		{
			name:     "No guide",
			comments: []string{"Some comment", "Another comment"},
			expected: false,
		},
		{
			name:     "Empty comments",
			comments: []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasExistingGuide(tt.comments)
			if result != tt.expected {
				t.Errorf("HasExistingGuide(%v) = %v, want %v",
					tt.comments, result, tt.expected)
			}
		})
	}
}
