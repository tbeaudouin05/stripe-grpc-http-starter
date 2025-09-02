package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stripe/stripe-go"
	"github.com/tbeaudouin05/stripe-trellai/api/config"
	stripedb "github.com/tbeaudouin05/stripe-trellai/api/services/stripe/db"
)

// VerifySubscription checks if a subscription is valid for a given user external id
func (s serviceImpl) VerifySubscription(userExternalID string) (VerifySubscriptionResponse, error) {
	// if there is enough free credit, then it is valid
	credit, err := stripedb.GetFreeCredit(userExternalID)
	if err != nil {
		return VerifySubscriptionResponse{}, fmt.Errorf("%w: error retrieving free credit: %v", ErrDatabase, err)
	}
	if credit > 0 {
		return VerifySubscriptionResponse{IsValidSubscription: true, ValidityType: ValidityTypeFreeTier}, nil
	}

	// if not enough free credit, fetch user account and customer ID
	ua, err := stripedb.GetUserAccount(userExternalID)
	if err != nil {
		return VerifySubscriptionResponse{}, fmt.Errorf("%w: error retrieving user account: %v", ErrDatabase, err)
	}
	if ua.UserExternalID == stripedb.AccountWithoutSubscriptionID {
		return VerifySubscriptionResponse{IsValidSubscription: false, InvalidityType: InvalidityTypeNoSubscription}, nil
	}
	if ua.StripeSubscriptionID == "" {
		return VerifySubscriptionResponse{IsValidSubscription: false, InvalidityType: InvalidityTypeNoSubscription}, nil
	}

	// fetch subscription
	subRetrieved, err := s.gw.GetSubscription(ua.StripeSubscriptionID)
	if err != nil {
		return VerifySubscriptionResponse{}, fmt.Errorf("%w: error getting subscription: %v", ErrGateway, err)
	}

	// get customer email
	cust, err := s.gw.GetCustomer(ua.StripeCustomerID)
	if err != nil {
		return VerifySubscriptionResponse{}, fmt.Errorf("%w: error retrieving customer email: %v", ErrGateway, err)
	}
	email := cust.Email

	// if subscription is cancelled, then it is not valid
	if IsSubscriptionCancelled(subRetrieved) {
		return VerifySubscriptionResponse{IsValidSubscription: false, InvalidityType: InvalidityTypeCancelled, StripeCustomerEmail: email}, nil
	}

	// if subscription is not valid, then it is not valid :)
	if subRetrieved.Status != stripe.SubscriptionStatusActive {
		return VerifySubscriptionResponse{IsValidSubscription: false, InvalidityType: InvalidityTypeOther, StripeCustomerEmail: email}, nil
	}

	// if subscription is exhausted (not enough units remaining), then it is not valid
	// Stripe provides seconds; our DB stores milliseconds, so convert bounds to ms.
	count, err := stripedb.CountUnitsBetween(
		userExternalID,
		subRetrieved.CurrentPeriodStart*1000,
		subRetrieved.CurrentPeriodEnd*1000,
	)
	if err != nil {
		return VerifySubscriptionResponse{}, fmt.Errorf("%w: error counting units: %v", ErrDatabase, err)
	}
	if subRetrieved.Plan == nil {
		return VerifySubscriptionResponse{}, fmt.Errorf("plan not found for subscription")
	}
	if subRetrieved.Plan.Amount == 0 {
		return VerifySubscriptionResponse{}, fmt.Errorf("plan amount is 0 for subscription")
	}
	if subRetrieved.Quantity == 0 {
		return VerifySubscriptionResponse{}, fmt.Errorf("quantity is 0 for subscription")
	}

	// parse CREDIT_UNITS_PER_DOLLAR from config (underscores allowed, e.g., 2_000_000)
	if config.AppConfig == nil {
		return VerifySubscriptionResponse{}, fmt.Errorf("app config not initialized")
	}
	raw := strings.ReplaceAll(config.AppConfig.CreditUnitsPerDollar, "_", "")
	unitsPerDollar, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || unitsPerDollar <= 0 {
		return VerifySubscriptionResponse{}, fmt.Errorf("invalid CREDIT_UNITS_PER_DOLLAR: %q", config.AppConfig.CreditUnitsPerDollar)
	}
	// Stripe Amount is in cents; convert to dollars before multiplying
	dollars := subRetrieved.Plan.Amount / 100
	if int64(count) > dollars*subRetrieved.Quantity*unitsPerDollar {
		return VerifySubscriptionResponse{IsValidSubscription: false, InvalidityType: InvalidityTypeExhausted, StripeCustomerEmail: email}, nil
	}

	return VerifySubscriptionResponse{
		IsValidSubscription: true,
		ValidityType:        ValidityTypePayingCustomer,
		StripeCustomerEmail: email,
	}, nil
}

// CancelSubscription attempts to cancel a Stripe subscription by its ID.
func (s serviceImpl) CancelSubscription(subscriptionID string) error {
	return s.gw.CancelSubscription(subscriptionID)
}
