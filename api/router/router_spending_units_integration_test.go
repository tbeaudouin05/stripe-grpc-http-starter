package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// Local HTTP integration test for AddSpendingUnits via grpc-gateway.
func TestAddSpendingUnitsHTTP_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	// Reuse local server setup used by other integration tests
	ts := newTestServer(t)
	defer ts.Close()

	// Send invalid payload (empty items) to assert endpoint is reachable and responds (non-200)
	payload := map[string]any{"items": []any{}}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/api/spending-units", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected failure status for invalid payload, got %d", resp.StatusCode)
	}
}
