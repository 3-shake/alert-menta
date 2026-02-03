package firstresponse

import (
	"strings"
)

// SeverityLevel represents incident severity
type SeverityLevel string

const (
	SeverityHigh   SeverityLevel = "high"
	SeverityMedium SeverityLevel = "medium"
	SeverityLow    SeverityLevel = "low"
)

// SeverityEmoji returns an emoji for the severity level
func SeverityEmoji(severity SeverityLevel) string {
	switch severity {
	case SeverityHigh:
		return "🔴"
	case SeverityMedium:
		return "🟡"
	case SeverityLow:
		return "🟢"
	default:
		return "⚪"
	}
}

// DetermineSeverity determines the severity from issue labels and body
func DetermineSeverity(labels []string, body string) SeverityLevel {
	// 1. Check labels first (explicit severity)
	for _, label := range labels {
		labelLower := strings.ToLower(label)

		// Check for severity prefix patterns
		if strings.HasPrefix(labelLower, "severity:") ||
			strings.HasPrefix(labelLower, "sev:") ||
			strings.HasPrefix(labelLower, "priority:") {
			parts := strings.SplitN(labelLower, ":", 2)
			if len(parts) == 2 {
				return parseSeverityValue(strings.TrimSpace(parts[1]))
			}
		}

		// Check for direct severity labels
		switch labelLower {
		case "critical", "sev1", "p0", "high", "urgent":
			return SeverityHigh
		case "major", "sev2", "p1", "medium", "warning":
			return SeverityMedium
		case "minor", "sev3", "sev4", "p2", "p3", "low":
			return SeverityLow
		}
	}

	// 2. Infer from body content
	bodyLower := strings.ToLower(body)

	// High severity indicators
	highIndicators := []string{
		"production", "本番", "全ユーザー", "all users",
		"service down", "サービス停止", "outage", "障害",
		"data loss", "データ損失", "security", "セキュリティ",
		"緊急", "urgent", "critical",
	}
	for _, indicator := range highIndicators {
		if strings.Contains(bodyLower, indicator) {
			return SeverityHigh
		}
	}

	// Low severity indicators
	lowIndicators := []string{
		"staging", "ステージング", "development", "開発",
		"test", "テスト", "minor", "cosmetic",
	}
	for _, indicator := range lowIndicators {
		if strings.Contains(bodyLower, indicator) {
			return SeverityLow
		}
	}

	// Default to medium
	return SeverityMedium
}

// parseSeverityValue parses a severity value string
func parseSeverityValue(value string) SeverityLevel {
	switch strings.ToLower(value) {
	case "high", "critical", "1", "sev1", "p0":
		return SeverityHigh
	case "low", "minor", "3", "4", "sev3", "sev4", "p2", "p3":
		return SeverityLow
	default:
		return SeverityMedium
	}
}
