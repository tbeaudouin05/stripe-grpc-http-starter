package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	config "github.com/tbeaudouin05/stripe-trellai/api/config"
)

// Remote HTTP integration tests against the deployed gateway.
// Default base URL: https://ai-mails-backend.fly.dev

func remoteBaseURL() string {
	if config.AppConfig != nil && config.AppConfig.IntegrationBaseURL != "" {
		return config.AppConfig.IntegrationBaseURL
	}
	return "https://ai-mails-backend.fly.dev"
}

func TestCancelSubscriptionHTTP_Remote_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ensureConfig(t)
	base := remoteBaseURL()

	payload := map[string]any{"subscriptionId": ""}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(base+"/api/cancel-subscription", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 for invalid payload, got %d", resp.StatusCode)
	}
}

func TestVerifySubscriptionValidityHTTP_Remote_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ensureConfig(t)
	base := remoteBaseURL()

	payload := map[string]any{"userExternalId": ""}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(base+"/api/verify-subscription-validity", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 for invalid payload, got %d", resp.StatusCode)
	}
}

func TestReceiveStripeWebhookHTTP_Remote_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ensureConfig(t)
	base := remoteBaseURL()

	req, _ := http.NewRequest(http.MethodPost, base+"/api/receive-stripe-webhook", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	// Intentionally omit Stripe-Signature header to get an error response
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected non-200 when missing Stripe-Signature, got %d", resp.StatusCode)
	}
}
