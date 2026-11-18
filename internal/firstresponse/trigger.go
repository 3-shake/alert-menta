package firstresponse

import (
	"strings"
)

// ShouldTrigger checks if the first response guide should be triggered
func ShouldTrigger(issueLabels []string, triggerLabels []string) bool {
	if len(triggerLabels) == 0 {
		// No trigger labels configured, don't trigger
		return false
	}

	// Check if any of the trigger labels match
	for _, triggerLabel := range triggerLabels {
		triggerLower := strings.ToLower(triggerLabel)
		for _, issueLabel := range issueLabels {
			if strings.ToLower(issueLabel) == triggerLower {
				return true
			}
		}
	}

	return false
}

// HasExistingGuide checks if a guide comment already exists
func HasExistingGuide(comments []string) bool {
	guideMarkers := []string{
		"## 🚨 インシデント対応ガイド",
		"## Incident Response Guide",
		"<!-- alert-menta:first-response -->",
	}

	for _, comment := range comments {
		for _, marker := range guideMarkers {
			if strings.Contains(comment, marker) {
				return true
			}
		}
	}

	return false
}
