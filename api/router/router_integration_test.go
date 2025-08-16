package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	bootstrap "github.com/tbeaudouin05/stripe-trellai/api/bootstrap"
	config "github.com/tbeaudouin05/stripe-trellai/api/config"
)

func ensureConfig(t *testing.T) {
	t.Helper()
	if config.AppConfig == nil {
		cfg, err := config.LoadConfig()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		config.AppConfig = cfg
	}
}

func newTestServer(t *testing.T) *httptest.Server {
	// Use real bootstrap and services; router itself calls bootstrap.Ensure.
	ensureConfig(t)
	if err := bootstrap.Ensure(); err != nil {
		t.Fatalf("bootstrap ensure failed: %v", err)
	}
	h := NewRouter()
	return httptest.NewServer(h)
}

func TestCancelSubscriptionHTTP_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ts := newTestServer(t)
	defer ts.Close()

	// Send invalid/empty subscriptionId to assert endpoint responds (non-200)
	payload := map[string]any{"subscriptionId": ""}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/api/cancel-subscription", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected failure status for invalid payload, got %d", resp.StatusCode)
	}
}

func TestVerifySubscriptionValidityHTTP_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ts := newTestServer(t)
	defer ts.Close()

	// Send empty userExternalId to assert endpoint responds (non-200)
	payload := map[string]any{"userExternalId": ""}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/api/verify-subscription-validity", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected failure status for invalid payload, got %d", resp.StatusCode)
	}
}

func TestReceiveStripeWebhookHTTP_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ts := newTestServer(t)
	defer ts.Close()

	// No Stripe-Signature header on purpose – should fail
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+"/api/receive-stripe-webhook", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected failure status when missing Stripe-Signature, got %d", resp.StatusCode)
	}
}
