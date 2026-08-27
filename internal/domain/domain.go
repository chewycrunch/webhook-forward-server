// Package domain holds the core types shared across the store, queue,
// dispatcher, and handlers.
//
// Versioned wire formats do NOT belong here. A v1 request or response struct
// lives in internal/httpapi/v1, so that adding a field for v2 can never
// change what v1 already emits.
package domain

import "time"

// Endpoint is one forwarding destination: the caller-facing ID and secret
// from the send URL, mapped to the Discord webhook they forward to.
type Endpoint struct {
	// ID is the public identifier in the send URL. Safe to log.
	ID int64

	// TokenHash is the SHA-256 of the URL's token. The token itself is
	// never stored, so a database leak does not hand over live endpoints.
	TokenHash []byte

	// DiscordURL is the full destination webhook URL. Secret.
	DiscordURL string

	// DiscordWebhookID is parsed out of DiscordURL and is the key for the
	// rate limit bucket. Two Endpoints pointing at the same Discord webhook
	// must share one bucket, so this — not ID — keys the workers.
	DiscordWebhookID string

	CreatedAt time.Time
}

// Job is a single queued payload awaiting delivery to Discord.
type Job struct {
	ID         string
	EndpointID int64

	// WebhookID is Discord's webhook ID, copied from the Endpoint at
	// enqueue time. It is the rate limit bucket key and decides which
	// worker drains this job.
	WebhookID string

	// DestinationURL is the full Discord webhook URL, denormalized onto
	// the job so the dispatcher needs only the queue and an HTTP client.
	// It also freezes delivery against a later endpoint edit. Secret.
	DestinationURL string

	// Payload is the raw JSON body as received, forwarded verbatim.
	Payload []byte

	// Attempts counts delivery tries so far, used for backoff and for
	// giving up on a job that will never succeed.
	Attempts int

	EnqueuedAt time.Time

	// NotBefore is the earliest time to attempt delivery. Backoff and
	// 429 retry-after both work by pushing this forward.
	NotBefore time.Time
}
