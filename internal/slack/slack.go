package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client handles Slack webhook notifications
type Client struct {
	WebhookURL string
	Channel    string
	HTTPClient *http.Client
}

// NewClient creates a new Slack client
func NewClient(webhookURL, channel string) *Client {
	return &Client{
		WebhookURL: webhookURL,
		Channel:    channel,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Message represents a Slack message
type Message struct {
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment represents a Slack message attachment
type Attachment struct {
	Color     string   `json:"color"`
	Title     string   `json:"title,omitempty"`
	TitleLink string   `json:"title_link,omitempty"`
	Text      string   `json:"text"`
	Footer    string   `json:"footer,omitempty"`
	Fields    []Field  `json:"fields,omitempty"`
	MrkdwnIn  []string `json:"mrkdwn_in,omitempty"`
}

// Field represents a field in a Slack attachment
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SendCommandResponse sends a notification about a command execution result
func (c *Client) SendCommandResponse(issueTitle, issueURL, command, response string) error {
	// Truncate response if too long for Slack
	maxLen := 2000
	displayResponse := response
	if len(response) > maxLen {
		displayResponse = response[:maxLen] + "\n\n... (truncated, see full response in GitHub Issue)"
	}

	msg := Message{
		Text: fmt.Sprintf("🤖 alert-menta `/%s` command executed", command),
		Attachments: []Attachment{
			{
				Color:     "#36a64f",
				Title:     issueTitle,
				TitleLink: issueURL,
				Text:      displayResponse,
				Footer:    "alert-menta | GitHub Issue",
				MrkdwnIn:  []string{"text"},
			},
		},
	}

	if c.Channel != "" {
		msg.Channel = c.Channel
	}

	return c.send(msg)
}

// SendIncidentNotification sends a notification about a new incident
func (c *Client) SendIncidentNotification(issueTitle, issueURL, summary string) error {
	msg := Message{
		Text: "🚨 New Incident Created",
		Attachments: []Attachment{
			{
				Color:     "danger",
				Title:     issueTitle,
				TitleLink: issueURL,
				Text:      summary,
				Footer:    "alert-menta",
				MrkdwnIn:  []string{"text"},
			},
		},
	}

	if c.Channel != "" {
		msg.Channel = c.Channel
	}

	return c.send(msg)
}

// send sends a message to Slack via webhook
func (c *Client) send(msg Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}
