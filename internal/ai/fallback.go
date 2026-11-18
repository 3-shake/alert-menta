package ai

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// FallbackClient wraps multiple AI clients and tries them in order
type FallbackClient struct {
	clients    []Ai
	names      []string
	maxRetries int
	delayMs    int
	logger     *log.Logger
}

// FallbackClientConfig holds configuration for creating a FallbackClient
type FallbackClientConfig struct {
	MaxRetries int
	DelayMs    int
	Logger     *log.Logger
}

// NewFallbackClient creates a new FallbackClient with the given clients
func NewFallbackClient(clients []Ai, names []string, config FallbackClientConfig) *FallbackClient {
	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	delayMs := config.DelayMs
	if delayMs <= 0 {
		delayMs = 1000
	}

	return &FallbackClient{
		clients:    clients,
		names:      names,
		maxRetries: maxRetries,
		delayMs:    delayMs,
		logger:     config.Logger,
	}
}

// GetResponse tries each client in order until one succeeds
func (f *FallbackClient) GetResponse(prompt *Prompt) (string, error) {
	var allErrors []string

	for i, client := range f.clients {
		providerName := f.names[i]

		for retry := 0; retry < f.maxRetries; retry++ {
			if f.logger != nil {
				if retry > 0 {
					f.logger.Printf("Retry %d/%d for provider %s", retry+1, f.maxRetries, providerName)
				} else {
					f.logger.Printf("Trying provider: %s", providerName)
				}
			}

			response, err := client.GetResponse(prompt)
			if err == nil {
				if f.logger != nil && i > 0 {
					f.logger.Printf("Successfully got response from fallback provider: %s", providerName)
				}
				return response, nil
			}

			errMsg := fmt.Sprintf("%s (attempt %d): %v", providerName, retry+1, err)
			allErrors = append(allErrors, errMsg)

			if f.logger != nil {
				f.logger.Printf("Provider %s failed: %v", providerName, err)
			}

			// Check if error is retryable
			if !isRetryableError(err) {
				if f.logger != nil {
					f.logger.Printf("Error is not retryable, moving to next provider")
				}
				break
			}

			// Wait before retry (but not after last retry)
			if retry < f.maxRetries-1 {
				time.Sleep(time.Duration(f.delayMs) * time.Millisecond)
			}
		}
	}

	return "", fmt.Errorf("all providers failed: %s", strings.Join(allErrors, "; "))
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Retryable conditions
	retryablePatterns := []string{
		"timeout",
		"deadline exceeded",
		"connection refused",
		"connection reset",
		"temporary failure",
		"rate limit",
		"429", // Too Many Requests
		"500", // Internal Server Error
		"502", // Bad Gateway
		"503", // Service Unavailable
		"504", // Gateway Timeout
		"server error",
		"service unavailable",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}
