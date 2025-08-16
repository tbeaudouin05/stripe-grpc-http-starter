package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/joho/godotenv"
)

// AppConfig holds the global application configuration
var AppConfig *Config

// Config holds the application configuration
type Config struct {
	DatabaseURL         string
	StripeSecretKey     string
	StripeWebhookSecret string
	// Optional: base URL for running remote HTTP integration tests (e.g., https://api.example.com)
	IntegrationBaseURL  string
	// Server ports
	HTTPPort            string
	GRPCPort            string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{}

	// Try to load .env file from current directory and parent directories
	currentDir, _ := os.Getwd()
	for currentDir != "/" {
		// Check if .env file exists in current directory
		envPath := filepath.Join(currentDir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			// Load .env file
			err = godotenv.Load(envPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load .env file: %v", err)
			}
			break
		}
		// Move up one directory
		currentDir = filepath.Dir(currentDir)
	}

	// Get required environment variables
	requiredVars := []struct {
		name     string
		envVar   string
		display  string
		required bool
	}{
		{"DatabaseURL", "DATABASE_URL", "Database URL", true},
		{"StripeSecretKey", "STRIPE_SECRET_KEY", "Stripe Secret Key", true},
		{"StripeWebhookSecret", "STRIPE_WEBHOOK_SECRET", "Stripe Webhook Secret", true},
		// Optional integration base URL for remote tests
		{"IntegrationBaseURL", "INTEGRATION_BASE_URL", "Integration Base URL", false},
		// Optional server ports
		{"HTTPPort", "PORT", "HTTP Port", false},
		{"GRPCPort", "GRPC_PORT", "gRPC Port", false},
	}

	for _, v := range requiredVars {
		value := os.Getenv(v.envVar)
		if v.required && value == "" {
			return nil, fmt.Errorf("missing required environment variable: %s", v.display)
		}
		configField := reflect.ValueOf(config).Elem().FieldByName(v.name)
		configField.SetString(value)
	}

	// Defaults
	if config.HTTPPort == "" {
		config.HTTPPort = "8080"
	}
	if config.GRPCPort == "" {
		config.GRPCPort = "50051"
	}

	return config, nil
}
