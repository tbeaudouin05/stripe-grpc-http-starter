package bootstrap

import (
    "fmt"
    "sync"

    stripeapp "github.com/tbeaudouin05/stripe-trellai/api/services/stripe/app"
    stripegw "github.com/tbeaudouin05/stripe-trellai/api/services/stripe/gateway/stripe"
    "github.com/tbeaudouin05/stripe-trellai/api/config"
    "github.com/tbeaudouin05/stripe-trellai/api/database"
)

var stripeService stripeapp.Service
var initOnce sync.Once
var initErr error

// Init initializes config, database, and third-party clients, and wires services.
func Init() error {
    // If a service has already been injected (e.g., tests), do not override or init heavy deps.
    if stripeService != nil {
        return nil
    }
    var err error
    if config.AppConfig == nil {
        config.AppConfig, err = config.LoadConfig()
        if err != nil {
            return fmt.Errorf("failed to load config: %w", err)
        }
    }

    if err := database.Initialize(); err != nil {
        return fmt.Errorf("failed to initialize database: %w", err)
    }

    stripegw.SetKey(config.AppConfig.StripeSecretKey)

    stripeService = stripeapp.NewService(stripegw.New())
    return nil
}

func GetStripeService() stripeapp.Service { return stripeService }

// SetStripeService allows tests to inject a stub implementation.
func SetStripeService(s stripeapp.Service) { stripeService = s }

// Ensure runs Init() once per process and returns any initialization error.
func Ensure() error {
    initOnce.Do(func() {
        initErr = Init()
    })
    return initErr
}
