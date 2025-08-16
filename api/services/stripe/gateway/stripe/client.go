package stripegw

import (
    stripe "github.com/stripe/stripe-go"
    "github.com/stripe/stripe-go/customer"
    "github.com/stripe/stripe-go/sub"

    gw "github.com/tbeaudouin05/stripe-trellai/api/services/stripe/gateway"
)

// SetKey configures the Stripe SDK key once during bootstrap.
func SetKey(key string) { stripe.Key = key }

// client is the Stripe SDK-backed implementation of the gateway.
type client struct{}

// New returns a StripeGateway backed by the official Stripe SDK.
func New() gw.StripeGateway { return client{} }

func (client) GetSubscription(id string) (stripe.Subscription, error) {
    subPtr, err := sub.Get(id, nil)
    if err != nil {
        return stripe.Subscription{}, err
    }
    if subPtr == nil {
        return stripe.Subscription{}, nil
    }
    return *subPtr, nil
}

func (client) CancelSubscription(id string) error {
    _, err := sub.Cancel(id, nil)
    return err
}

func (client) GetCustomer(id string) (stripe.Customer, error) {
    custPtr, err := customer.Get(id, nil)
    if err != nil {
        return stripe.Customer{}, err
    }
    if custPtr == nil {
        return stripe.Customer{}, nil
    }
    return *custPtr, nil
}
