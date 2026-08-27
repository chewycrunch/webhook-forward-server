// Package discord is the outbound HTTP client. It owns one job: perform a
// single delivery attempt and report back what the response said about the
// rate limit. Pacing decisions belong to the dispatcher.
package discord

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Result describes the outcome of one delivery attempt.
type Result struct {
	StatusCode int

	// Remaining and ResetAfter come from the X-RateLimit-* headers. They are
	// the authoritative pacing signal: Discord's actual limits are
	// undocumented and change, so never hardcode a rate.
	Remaining  int
	ResetAfter time.Duration

	// RetryAfter is set on a 429 and outranks ResetAfter.
	RetryAfter time.Duration

	// Global reports X-RateLimit-Scope: global, meaning back off every
	// destination rather than just this bucket.
	Global bool
}

// Retryable reports whether another attempt could plausibly succeed. A 4xx
// other than 429 never will, and retrying it forever both poisons the queue
// and walks us into Discord's invalid-request ban.
func (r Result) Retryable() bool {
	return r.StatusCode == http.StatusTooManyRequests || r.StatusCode >= 500
}

// Client sends payloads to Discord webhook URLs.
type Client struct {
	http *http.Client
}

func New(timeout time.Duration) *Client {
	return &Client{http: &http.Client{Timeout: timeout}}
}

// Send performs one delivery attempt. A non-2xx response is reported through
// Result, not through err; err is reserved for transport failures.
func (c *Client) Send(ctx context.Context, url string, payload []byte) (Result, error) {
	// TODO: POST payload as application/json, then parse the rate limit
	// headers and any 429 body into a Result.
	return Result{}, errors.New("discord: send not implemented")
}
