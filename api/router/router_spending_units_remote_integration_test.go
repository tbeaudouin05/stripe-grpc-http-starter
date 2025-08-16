package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	config "github.com/tbeaudouin05/stripe-trellai/api/config"
)

// This test targets a remote deployment via grpc-gateway HTTP.
// It is skipped by default unless INTEGRATION_BASE_URL is provided.
func TestAddSpendingUnitsHTTP_Remote_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	ensureConfig(t)
	base := config.AppConfig.IntegrationBaseURL
	if base == "" {
		// Default to the Fly deployment base URL when not provided via env
		base = "https://ai-mails-backend.fly.dev"
	}

	// Send invalid payload (empty items) to assert endpoint is reachable and responds (non-200)
	payload := map[string]any{"items": []any{}}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(base+"/api/spending-units", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected failure status for invalid payload, got %d", resp.StatusCode)
	}
}
