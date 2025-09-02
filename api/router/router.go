package router

import (
    "context"
    "io"
    "log/slog"
    "net/http"

    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    bootstrap "github.com/tbeaudouin05/stripe-trellai/api/bootstrap"
    config "github.com/tbeaudouin05/stripe-trellai/api/config"
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

    // Register a raw HTTP handler for Stripe webhooks to preserve exact body bytes for signature verification.
    if err := mux.HandlePath(http.MethodPost, "/api/receive-stripe-webhook", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
        if err := bootstrap.Ensure(); err != nil {
            slog.Error("bootstrap ensure failed", "err", err)
            http.Error(w, "initialization error", http.StatusInternalServerError)
            return
        }
        sig := r.Header.Get("Stripe-Signature")
        if sig == "" {
            http.Error(w, "missing Stripe-Signature header", http.StatusBadRequest)
            return
        }
        body, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, "failed to read body", http.StatusBadRequest)
            return
        }
        event, err := grpcserver.ConstructEvent(body, sig, config.AppConfig.StripeWebhookSecret)
        if err != nil {
            slog.Error("webhook signature verification failed", "err", err)
            http.Error(w, "signature verification failed", http.StatusBadRequest)
            return
        }
        switch event.Type {
        case "checkout.session.completed":
            if err := bootstrap.GetStripeService().HandleCheckoutSessionCompleted(event); err != nil {
                slog.Error("handle checkout.session.completed failed", "err", err)
                http.Error(w, "handler error", http.StatusInternalServerError)
                return
            }
        default:
            slog.Info("Unhandled event type", "type", event.Type)
        }
        w.WriteHeader(http.StatusOK)
    }); err != nil {
        slog.Error("failed to register raw webhook handler", "err", err)
    }
    return mux
}
