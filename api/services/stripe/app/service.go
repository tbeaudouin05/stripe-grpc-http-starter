package app

import (
    "encoding/json"
    "fmt"
    "log/slog"

    stripe "github.com/stripe/stripe-go"
    stripedb "github.com/tbeaudouin05/stripe-trellai/api/services/stripe/db"
    gw "github.com/tbeaudouin05/stripe-trellai/api/services/stripe/gateway"
)

// Service defines the business operations for the Stripe domain.
// Implementation uses shared database package directly for now (Phase 1).
type Service interface {
    CancelSubscription(subscriptionID string) error
    VerifySubscription(userExternalID string) (VerifySubscriptionResponse, error)
    HandleCheckoutSessionCompleted(event stripe.Event) error
    AddSpendingUnits(items []stripedb.SpendingUnit) (int, error)
}

// serviceImpl is a concrete implementation.
// No fields needed yet since we rely on package-level database funcs.
type serviceImpl struct{ gw gw.StripeGateway }

func NewService(g gw.StripeGateway) Service { return serviceImpl{gw: g} }

// HandleCheckoutSessionCompleted processes the checkout.session.completed event
func (s serviceImpl) HandleCheckoutSessionCompleted(event stripe.Event) error {
    slog.Info("HandleCheckoutSessionCompleted: start", "event_type", event.Type, "event_id", event.ID)
    var session stripe.CheckoutSession
    if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
        slog.Error("error unmarshaling into CheckoutSession", "err", err)
        return fmt.Errorf("%w: error unmarshaling into CheckoutSession: %v", ErrBadEvent, err)
    }
    if session.ClientReferenceID == "" {
        slog.Error("client reference ID not found in CheckoutSession")
        return fmt.Errorf("%w: client reference ID not found in CheckoutSession", ErrBadEvent)
    }
    if session.Customer == nil || session.Customer.ID == "" {
        slog.Error("customer ID not found in CheckoutSession")
        return fmt.Errorf("%w: customer ID not found in CheckoutSession", ErrBadEvent)
    }
    if session.Subscription == nil || session.Subscription.ID == "" {
        slog.Error("subscription ID not found in CheckoutSession")
        return fmt.Errorf("%w: subscription ID not found in CheckoutSession", ErrBadEvent)
    }
    userExternalID := session.ClientReferenceID
    stripeCustomerID := session.Customer.ID
    newStripeSubscriptionID := session.Subscription.ID

    exists, existingSubID, err := stripedb.CheckUserAccount(userExternalID)
    if err != nil {
        slog.Error("error checking user account", "user_external_id", userExternalID, "err", err)
        return fmt.Errorf("%w: %v", ErrDatabase, err)
    }
    if !exists {
        slog.Info("no existing user account", "user_external_id", userExternalID)
        if err := stripedb.UpsertUserAccount(userExternalID, newStripeSubscriptionID, "no_need", stripeCustomerID); err != nil {
            slog.Error("error upserting user_account", "user_external_id", userExternalID, "stripe_subscription_id", newStripeSubscriptionID, "stripe_customer_id", stripeCustomerID, "err", err)
            return fmt.Errorf("%w: error upserting user_account: %v", ErrDatabase, err)
        }
    } else {
        slog.Info("existing user account found", "user_external_id", userExternalID, "existing_subscription_id", existingSubID)
        // If the account exists but has no previous subscription ID, treat it as no previous subscription
        if existingSubID == "" {
            slog.Info("no previous subscription ID on existing account; upserting new subscription", "user_external_id", userExternalID)
            if err := stripedb.UpsertUserAccount(userExternalID, newStripeSubscriptionID, "no_need", stripeCustomerID); err != nil {
                slog.Error("error upserting user_account when no previous subscription id", "user_external_id", userExternalID, "stripe_subscription_id", newStripeSubscriptionID, "stripe_customer_id", stripeCustomerID, "err", err)
                return fmt.Errorf("%w: error upserting user_account: %v", ErrDatabase, err)
            }
        } else {
            prevSub, err := s.gw.GetSubscription(existingSubID)
            if err != nil {
                slog.Error("error fetching previous subscription", "existing_subscription_id", existingSubID, "err", err)
                return fmt.Errorf("error fetching previous subscription: %w", err)
            }
            if IsSubscriptionCancelled(prevSub) {
                slog.Info("previous subscription cancelled, replacing with new subscription", "user_external_id", userExternalID)
                if err := stripedb.UpsertUserAccount(userExternalID, newStripeSubscriptionID, "no_need", stripeCustomerID); err != nil {
                    slog.Error("error upserting user_account after previous subscription cancelled", "user_external_id", userExternalID, "stripe_subscription_id", newStripeSubscriptionID, "stripe_customer_id", stripeCustomerID, "err", err)
                    return fmt.Errorf("%w: error upserting user_account: %v", ErrDatabase, err)
                }
            } else {
                slog.Info("previous subscription active, recording new subscription as invalid", "user_external_id", userExternalID)
                if err := stripedb.InsertInvalidSubscription(userExternalID, newStripeSubscriptionID, "no_need", stripeCustomerID); err != nil {
                    slog.Error("error inserting invalid subscription", "user_external_id", userExternalID, "stripe_subscription_id", newStripeSubscriptionID, "stripe_customer_id", stripeCustomerID, "err", err)
                    return fmt.Errorf("%w: error inserting invalid subscription: %v", ErrDatabase, err)
                }
            }
        }
    }

    if _, err = stripedb.GetFreeCredit(userExternalID); err != nil {
        slog.Error("error initializing free credit", "user_external_id", userExternalID, "err", err)
        return fmt.Errorf("%w: error initializing free credit: %v", ErrDatabase, err)
    }
    return nil
}

// AddSpendingUnits inserts a batch of spending units and returns how many were inserted.
func (s serviceImpl) AddSpendingUnits(items []stripedb.SpendingUnit) (int, error) {
    n, err := stripedb.AddSpendingUnits(items)
    if err != nil {
        return 0, fmt.Errorf("%w: %v", ErrDatabase, err)
    }
    return n, nil
}
