// Package store persists endpoints: the mapping from a caller-facing ID and
// token to the Discord webhook they forward to.
package store

import (
	"context"
	"errors"

	"github.com/chewycrunch/webhook-forward-server/internal/domain"
)

// ErrNotFound means no endpoint exists with that ID. Callers must not
// distinguish this from a bad token in anything they return to the client,
// or the API becomes an endpoint-enumeration oracle.
var ErrNotFound = errors.New("store: endpoint not found")

// Store is the endpoint repository, backed by SQLite.
type Store struct {
	path string
}

// Open connects to the database at path and applies any pending migrations.
func Open(path string) (*Store, error) {
	// TODO: open sqlite, create the endpoints table if absent.
	return &Store{path: path}, nil
}

func (s *Store) Close() error {
	// TODO: close the underlying handle.
	return nil
}

// EndpointByID looks up a single endpoint. It returns the stored token hash,
// never a token; verification is the caller's job.
func (s *Store) EndpointByID(ctx context.Context, id int64) (domain.Endpoint, error) {
	// TODO: SELECT ... WHERE id = ?
	return domain.Endpoint{}, ErrNotFound
}

// Create registers a new endpoint and returns it with its assigned ID.
func (s *Store) Create(ctx context.Context, e domain.Endpoint) (domain.Endpoint, error) {
	// TODO: INSERT and return the generated ID.
	return domain.Endpoint{}, errors.New("store: create not implemented")
}
