// Package queue is the durable job queue sitting between the inbound API and
// the outbound dispatcher. Durability is the point: a deep backlog is this
// service's normal state, so a restart must not drop it.
package queue

import (
	"context"
	"errors"
	"time"

	"github.com/chewycrunch/webhook-forward-server/internal/domain"
)

// ErrEmpty means no job is ready to run right now. It is an expected result,
// not a failure: the dispatcher waits and asks again.
var ErrEmpty = errors.New("queue: no job ready")

// Queue is a SQLite-backed job queue.
type Queue struct {
	path string
}

// Open connects to the database at path and applies any pending migrations.
func Open(path string) (*Queue, error) {
	// TODO: open sqlite, create the jobs table if absent.
	return &Queue{path: path}, nil
}

func (q *Queue) Close() error {
	// TODO: close the underlying handle.
	return nil
}

// Enqueue durably records a job for later delivery.
func (q *Queue) Enqueue(ctx context.Context, job domain.Job) error {
	// TODO: INSERT into jobs.
	return errors.New("queue: enqueue not implemented")
}

// Depth reports how many jobs are queued for one endpoint, so the inbound
// side can shed load before one noisy caller starves everyone else.
func (q *Queue) Depth(ctx context.Context, endpointID int64) (int, error) {
	// TODO: SELECT COUNT(*) WHERE endpoint_id = ?
	return 0, nil
}

// Lease claims the next ready job for a destination webhook and hands it to
// exactly one worker. Returns ErrEmpty when nothing is due.
//
// Jobs are keyed by Discord webhook ID rather than by endpoint ID because
// Discord's rate limit bucket is per destination webhook: two endpoints
// pointing at the same webhook must drain through one worker.
func (q *Queue) Lease(ctx context.Context, webhookID string) (domain.Job, error) {
	// TODO: SELECT the oldest job with not_before <= now, mark it leased.
	return domain.Job{}, ErrEmpty
}

// Ack removes a job that was delivered successfully.
func (q *Queue) Ack(ctx context.Context, jobID string) error {
	// TODO: DELETE FROM jobs WHERE id = ?
	return errors.New("queue: ack not implemented")
}

// Nack returns a job to the queue, deferred until notBefore. This is the one
// mechanism behind both retry backoff and honouring a 429 retry-after.
func (q *Queue) Nack(ctx context.Context, jobID string, notBefore time.Time) error {
	// TODO: UPDATE jobs SET not_before = ?, attempts = attempts + 1.
	return errors.New("queue: nack not implemented")
}

// WebhookIDs lists the destinations with work waiting, so the dispatcher
// knows which workers to run after a restart.
func (q *Queue) WebhookIDs(ctx context.Context) ([]string, error) {
	// TODO: SELECT DISTINCT webhook_id FROM jobs.
	return nil, nil
}
