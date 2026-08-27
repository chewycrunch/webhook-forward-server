// Package v1 implements version 1 of the forwarding API. Its request and
// response shapes are frozen: callers paste send URLs into third-party tools
// and never update them, so this package may gain optional fields but must
// never change the meaning of an existing one.
//
// Everything version-specific lives here: wire structs, decoding, and the
// mapping from service errors to status codes. The business logic itself is
// in internal/forward and is shared with every other version.
package v1

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/chewycrunch/webhook-forward-server/internal/domain"
)

// Forwarder is the slice of the forward service these handlers use.
// Declared here, on the consumer side, so a test can drive the handlers with
// a fake instead of a real store and queue.
type Forwarder interface {
	Enqueue(ctx context.Context, endpointID int64, token string, payload []byte) (domain.Job, error)
}

// API holds the resources the v1 handlers need.
type API struct {
	log *slog.Logger
	fwd Forwarder
}

// NewRouter registers the v1 routes. Paths are relative to the mount point,
// so "POST /send/..." is served at /api/v1/send/...
func NewRouter(log *slog.Logger, fwd Forwarder) http.Handler {
	api := &API{log: log, fwd: fwd}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /send/{endpointID}/{token}", api.handleSend)

	return mux
}
