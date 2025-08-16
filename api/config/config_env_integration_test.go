package config

import "testing"

// TestLoadConfig_Environment_Integration ensures required env vars are present
// in the deployment environment by invoking LoadConfig(). It is skipped in -short mode.
func TestLoadConfig_Environment_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping environment config test in -short mode")
	}
	if _, err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
}
