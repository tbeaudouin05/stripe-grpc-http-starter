package app

import "errors"

// Typed errors for the Stripe app layer. These enable HTTP mapping without
// relying on SDK-specific error types at the transport layer.
var (

	// ErrBadEvent indicates the incoming event payload is invalid or missing required fields.
	ErrBadEvent = errors.New("bad event")
	// ErrDatabase indicates a database-related failure.
	ErrDatabase = errors.New("database error")
	// ErrGateway indicates a failure from the Stripe gateway / API calls.
	ErrGateway = errors.New("gateway error")
)
