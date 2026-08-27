package v1

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/chewycrunch/webhook-forward-server/internal/forward"
)

// maxPayloadBytes bounds a single inbound payload. Without a cap, one caller
// can queue arbitrarily large bodies and exhaust disk.
const maxPayloadBytes = 1 << 20 // 1 MiB

// sendResponse is the v1 wire format. It lives here rather than in
// internal/domain precisely because it is versioned: v2 gets its own struct,
// so adding a field there can never change what v1 emits.
type sendResponse struct {
	JobID  string `json:"jobId"`
	Status string `json:"status"`
}

// errorResponse is the v1 error shape.
type errorResponse struct {
	Error string `json:"error"`
}

// handleSend queues a payload and returns immediately. Delivery to Discord
// happens later, on the worker that owns the destination's rate limit
// bucket, so this returns 202 rather than 200: nothing has been sent yet.
func (a *API) handleSend(w http.ResponseWriter, r *http.Request) {
	endpointID, err := strconv.ParseInt(r.PathValue("endpointID"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "unknown endpoint"})
		return
	}

	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxPayloadBytes))
	if err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "payload too large"})
		return
	}

	job, err := a.fwd.Enqueue(r.Context(), endpointID, r.PathValue("token"), payload)

	// Translating service errors into status codes is a per-version concern.
	// v2 may answer these same errors differently without touching forward.
	switch {
	case errors.Is(err, forward.ErrUnauthorized):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "unknown endpoint"})
		return
	case errors.Is(err, forward.ErrQueueFull):
		writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: "queue full"})
		return
	case err != nil:
		a.log.Error("enqueue failed", "err", err, "endpoint_id", endpointID)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		return
	}

	writeJSON(w, http.StatusAccepted, sendResponse{JobID: job.ID, Status: "queued"})
}
