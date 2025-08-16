package gateway

import stripe "github.com/stripe/stripe-go"

// StripeGateway abstracts Stripe SDK operations needed by the app layer.
// Methods return values (not pointers) to respect the project's preference
// to avoid pointer types in public interfaces.
type StripeGateway interface {
    GetSubscription(id string) (stripe.Subscription, error)
    CancelSubscription(id string) error
    GetCustomer(id string) (stripe.Customer, error)
}
