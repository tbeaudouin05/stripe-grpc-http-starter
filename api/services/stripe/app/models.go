package app

type InvalidityType string

type ValidityType string

const (
    InvalidityTypeNoSubscription InvalidityType = "noSubscription"
    InvalidityTypeCancelled      InvalidityType = "cancelled"
    InvalidityTypeExhausted      InvalidityType = "exhausted"
    InvalidityTypeOther          InvalidityType = "other"
)

const (
    ValidityTypeFreeTier       ValidityType = "freeTier"
    ValidityTypePayingCustomer ValidityType = "payingCustomer"
)

// Business constants
const PricePerTicket = 35

// VerifySubscriptionResponse is the domain response returned by the app layer
// HTTP layer will translate this into JSON
// Keep value types to avoid pointer proliferation in domain.
type VerifySubscriptionResponse struct {
    IsValidSubscription bool           `json:"isValidSubscription"`
    InvalidityType      InvalidityType `json:"invalidityType"`
    ValidityType        ValidityType   `json:"validityType"`
    StripeCustomerEmail string         `json:"stripeCustomerEmail"`
}
