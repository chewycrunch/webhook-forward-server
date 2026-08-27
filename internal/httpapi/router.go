// Package httpapi is the HTTP delivery layer. This file owns only the
// top-level routing table; every request handler lives in a version
// subpackage so that v1 can be frozen once it ships.
package httpapi

import (
	"log/slog"
	"net/http"

	v1 "github.com/chewycrunch/webhook-forward-server/internal/httpapi/v1"
)

// NewRouter builds the top-level mux and mounts each API version under its
// own prefix. Shared resources arrive as arguments and are passed down, so
// no package reaches for a global.
func NewRouter(log *slog.Logger, fwd v1.Forwarder) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealthz)

	// The prefix is stripped, so v1 registers routes relative to /api/v1
	// and never has to know where it is mounted.
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", v1.NewRouter(log, fwd)))

	return logRequests(log, mux)
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
