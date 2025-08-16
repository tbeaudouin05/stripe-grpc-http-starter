package app

import (
    "time"

    "github.com/stripe/stripe-go"
)

// IsSubscriptionCancelled returns true if the subscription is cancelled or past its cancel timestamp
func IsSubscriptionCancelled(sub stripe.Subscription) bool {
    now := time.Now().Unix()
    if sub.CancelAt != 0 && now > sub.CancelAt {
        return true
    }
    if sub.Status == stripe.SubscriptionStatusCanceled {
        return true
    }
    return false
}
