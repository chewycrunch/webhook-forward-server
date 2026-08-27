// Package forward is the inbound service layer: it authenticates a caller,
// decides whether to admit the payload, and queues it. It returns as soon as
// the job is durable and never waits on Discord.
//
// It returns domain errors, never HTTP status codes. Mapping those to
// responses belongs to each API version, which is what lets v1 and v2 share
// this logic while answering differently on the wire.
package forward

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/chewycrunch/webhook-forward-server/internal/domain"
	"github.com/chewycrunch/webhook-forward-server/internal/store"
)

var (
	// ErrUnauthorized covers both an unknown endpoint and a bad token.
	// They are deliberately indistinguishable so the API cannot be used to
	// enumerate which endpoint IDs exist.
	ErrUnauthorized = errors.New("forward: unauthorized")

	// ErrQueueFull means this endpoint is at its backlog limit.
	ErrQueueFull = errors.New("forward: queue full")
)

// EndpointStore is the slice of the store this package uses. Declared here,
// on the consumer side, so store stays unaware of who calls it.
type EndpointStore interface {
	EndpointByID(ctx context.Context, id int64) (domain.Endpoint, error)
}

// Queue is the inbound half of the job queue. The dispatcher declares its
// own, larger interface for the draining side.
type Queue interface {
	Enqueue(ctx context.Context, job domain.Job) error
	Depth(ctx context.Context, endpointID int64) (int, error)
}

// Service admits payloads onto the queue.
type Service struct {
	endpoints EndpointStore
	queue     Queue
	maxDepth  int
	log       *slog.Logger
}

func New(endpoints EndpointStore, queue Queue, maxDepth int, log *slog.Logger) *Service {
	return &Service{endpoints: endpoints, queue: queue, maxDepth: maxDepth, log: log}
}

// Enqueue authenticates the endpoint and durably queues the payload for
// later delivery.
func (s *Service) Enqueue(ctx context.Context, endpointID int64, token string, payload []byte) (domain.Job, error) {
	endpoint, err := s.endpoints.EndpointByID(ctx, endpointID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		// Collapse into ErrUnauthorized so a missing endpoint and a wrong
		// token are indistinguishable to the caller.
		return domain.Job{}, ErrUnauthorized
	case err != nil:
		return domain.Job{}, fmt.Errorf("looking up endpoint: %w", err)
	}

	if !validToken(token, endpoint.TokenHash) {
		return domain.Job{}, ErrUnauthorized
	}

	depth, err := s.queue.Depth(ctx, endpointID)
	if err != nil {
		return domain.Job{}, fmt.Errorf("reading queue depth: %w", err)
	}
	if depth >= s.maxDepth {
		return domain.Job{}, ErrQueueFull
	}

	id, err := newJobID()
	if err != nil {
		return domain.Job{}, fmt.Errorf("generating job id: %w", err)
	}

	now := time.Now().UTC()
	job := domain.Job{
		ID:             id,
		EndpointID:     endpoint.ID,
		WebhookID:      endpoint.DiscordWebhookID,
		DestinationURL: endpoint.DiscordURL,
		Payload:        payload,
		EnqueuedAt:     now,
		NotBefore:      now,
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		return domain.Job{}, fmt.Errorf("enqueueing job: %w", err)
	}

	s.log.LogAttrs(ctx, slog.LevelInfo, "job queued",
		slog.String("job_id", job.ID),
		slog.Int64("endpoint_id", job.EndpointID),
		slog.Int("depth", depth+1),
	)

	return job, nil
}

// validToken compares a presented token against the stored hash. The
// comparison is constant time: a byte-wise early exit would leak the hash
// one byte at a time to anyone able to measure response latency.
func validToken(token string, wantHash []byte) bool {
	got := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(got[:], wantHash) == 1
}

// HashToken produces the value to persist for an endpoint. The token itself
// is shown once at creation and never stored, so a database leak does not
// hand over live endpoints.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func newJobID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
