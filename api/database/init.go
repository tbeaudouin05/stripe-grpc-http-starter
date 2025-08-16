package database

import (
	"fmt"
	"log"

	config "github.com/tbeaudouin05/stripe-trellai/api/config"
)

func init() {
	// Initialize the database when the package is first imported
	// This assumes config.AppConfig has already been set in the main package
	if config.AppConfig == nil {
		var err error
		config.AppConfig, err = config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	}

	if err := Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	fmt.Println("Database initialized successfully")
}
