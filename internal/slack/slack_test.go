package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://hooks.slack.com/test", "#incidents")

	if client.WebhookURL != "https://hooks.slack.com/test" {
		t.Errorf("expected webhook URL to be set")
	}
	if client.Channel != "#incidents" {
		t.Errorf("expected channel to be #incidents")
	}
	if client.HTTPClient == nil {
		t.Errorf("expected HTTP client to be initialized")
	}
}

func TestSendCommandResponse(t *testing.T) {
	var receivedMessage Message

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json")
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedMessage); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "#test-channel")
	err := client.SendCommandResponse(
		"Test Issue",
		"https://github.com/test/repo/issues/1",
		"describe",
		"This is a test response",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMessage.Channel != "#test-channel" {
		t.Errorf("expected channel #test-channel, got %s", receivedMessage.Channel)
	}
	if len(receivedMessage.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(receivedMessage.Attachments))
	}
	if receivedMessage.Attachments[0].Title != "Test Issue" {
		t.Errorf("expected title 'Test Issue', got %s", receivedMessage.Attachments[0].Title)
	}
}

func TestSendIncidentNotification(t *testing.T) {
	var receivedMessage Message

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedMessage); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	err := client.SendIncidentNotification(
		"Production Outage",
		"https://github.com/test/repo/issues/2",
		"API server is down",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMessage.Text != "🚨 New Incident Created" {
		t.Errorf("unexpected message text: %s", receivedMessage.Text)
	}
	if receivedMessage.Attachments[0].Color != "danger" {
		t.Errorf("expected danger color for incident")
	}
}

func TestSendCommandResponse_TruncatesLongResponse(t *testing.T) {
	var receivedMessage Message

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedMessage); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")

	// Create a very long response
	longResponse := ""
	for i := 0; i < 300; i++ {
		longResponse += "This is a long response line. "
	}

	err := client.SendCommandResponse(
		"Test Issue",
		"https://github.com/test/repo/issues/1",
		"describe",
		longResponse,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedMessage.Attachments[0].Text) > 2100 {
		t.Errorf("response was not truncated properly")
	}
}

func TestSend_ErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	err := client.SendCommandResponse("Test", "http://test", "describe", "test")

	if err == nil {
		t.Errorf("expected error for non-200 response")
	}
}
