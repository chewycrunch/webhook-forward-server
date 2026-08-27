// Package dispatch is the outbound service layer: it drains the queue and
// delivers to Discord, paced by the rate limits Discord reports.
//
// It shares no call stack with the inbound side. The two halves communicate
// only through the queue, which is what allows this to run as a separate
// process later without changing either package.
package dispatch

import (
	"context"
	"log/slog"
	"time"

	"github.com/chewycrunch/webhook-forward-server/internal/discord"
	"github.com/chewycrunch/webhook-forward-server/internal/domain"
)

// Queue is the draining half of the job queue, declared here because it is a
// different and larger set of operations than the inbound side needs.
type Queue interface {
	Lease(ctx context.Context, webhookID string) (domain.Job, error)
	Ack(ctx context.Context, jobID string) error
	Nack(ctx context.Context, jobID string, notBefore time.Time) error
	WebhookIDs(ctx context.Context) ([]string, error)
}

// Dispatcher runs one worker per destination webhook. One worker per bucket
// gives serialized delivery and message ordering for free.
type Dispatcher struct {
	queue  Queue
	client *discord.Client
	log    *slog.Logger

	// globalLimit caps outbound requests per second across every worker.
	// Per-bucket pacing alone does not protect us: Discord limits per IP
	// too, so a thousand idle-but-ready buckets could still trip it.
	globalLimit float64
}

func New(queue Queue, client *discord.Client, globalLimit float64, log *slog.Logger) *Dispatcher {
	return &Dispatcher{queue: queue, client: client, globalLimit: globalLimit, log: log}
}

// Run drains the queue until ctx is cancelled. It blocks, so main starts it
// on its own goroutine.
func (d *Dispatcher) Run(ctx context.Context) error {
	d.log.Info("dispatcher started", "global_limit_rps", d.globalLimit)

	// TODO: resume a worker per queue.WebhookIDs, spawn one on first
	// enqueue for a new destination, and retire idle workers.
	<-ctx.Done()

	d.log.Info("dispatcher stopped")
	return nil
}

// deliver performs one attempt and applies the outcome to the queue: Ack on
// success, Nack with a computed delay on a retryable failure, and Ack on a
// permanent 4xx so a poison job cannot block its bucket forever.
func (d *Dispatcher) deliver(ctx context.Context, job domain.Job) error {
	// TODO: acquire a global token, send, then branch on Result.
	return nil
}
