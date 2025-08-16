package router

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	bootstrap "github.com/tbeaudouin05/stripe-trellai/api/bootstrap"
	grpcserver "github.com/tbeaudouin05/stripe-trellai/api/services/stripe/grpc"
)

// NewRouter returns the central HTTP router for the API using grpc-gateway.
// It maps the gRPC StripeService to HTTP endpoints using google.api.http options.
func NewRouter() http.Handler {
	// Initialize app dependencies (non-fatal if it fails here; RPCs re-check).
	if err := bootstrap.Ensure(); err != nil {
		slog.Error("bootstrap ensure failed", "err", err)
	}

	mux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(grpcserver.HeaderMatcher))
	srv := grpcserver.New(bootstrap.GetStripeService())
	if err := grpcserver.RegisterGateway(context.Background(), mux, srv); err != nil {
		slog.Error("failed to register grpc-gateway", "err", err)
	}
	return mux
}
