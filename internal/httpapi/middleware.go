package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// logRequests emits one line per request. It wraps the top-level mux, so
// every API version gets request logging without opting in.
func logRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(&rec, r)

		log.LogAttrs(r.Context(), slog.LevelInfo, "request",
			slog.String("method", r.Method),
			slog.String("path", redactPath(r.URL.Path)),
			slog.Int("status", rec.status),
			slog.Duration("took", time.Since(start)),
		)
	})
}

// statusRecorder captures the response status, which net/http does not
// expose after the handler has run. It defaults to 200 because a handler
// that writes a body without calling WriteHeader has implicitly sent one.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// redactPath strips the token from /api/{version}/send/{id}/{token}. The
// token is a live credential, so logging the raw path would scatter working
// endpoint secrets across every log sink we ship to. Matching on position
// rather than a literal prefix keeps this correct for future versions.
func redactPath(p string) string {
	seg := strings.Split(p, "/")
	if len(seg) >= 6 && seg[1] == "api" && seg[3] == "send" {
		seg[5] = "redacted"
		return strings.Join(seg[:6], "/")
	}
	return p
}
