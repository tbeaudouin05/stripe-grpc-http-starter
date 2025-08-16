package config

import (
	"log"
	"strings"
)

const (
	// ProdDbId is the identifier for the production database
	ProdDbId = "old-cloud"

	// InitialFreeCredit is the default number of free credits for new users
	InitialFreeCredit = 5
)

// CheckNotProdDB aborts immediately if the configured database URL contains ProdDbId.
// This should be called at the start of any test that interacts with the database.
func CheckNotProdDB() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DatabaseURL is not configured")
	}
	if strings.Contains(cfg.DatabaseURL, ProdDbId) {
		log.Fatalf("Tests aborted: DatabaseURL contains production identifier %s", ProdDbId)
	}
}
