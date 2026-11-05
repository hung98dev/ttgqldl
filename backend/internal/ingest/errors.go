package ingest

import (
	"errors"
)

// Stable rejection categories. Classification is carried by Decision; errors are used
// only for malformed input, capacity degradation and reservation lifecycle misuse.
var (
	ErrInvalidInput       = errors.New("ingest: invalid input")
	ErrInvalidAction      = errors.New("ingest: invalid parsed action")
	ErrInvalidIdentity    = errors.New("ingest: invalid or missing stable actor identity")
	ErrCacheFull          = errors.New("ingest: bounded in-memory cache is full; durable ingestion required")
	ErrReservationClosed  = errors.New("ingest: reservation already finalized or released")
	ErrReservationExpired = errors.New("ingest: reservation expired before finalization")
	ErrSessionNotFound    = errors.New("ingest: session not found")
	ErrSessionUnavailable = errors.New("ingest: session lifecycle does not accept durable input")
	ErrProviderMismatch   = errors.New("ingest: provider source does not match the session mode")
	ErrLiveBlocked        = errors.New("ingest: authorized_live is blocked by GATE-LIVE")
	ErrSequenceOverflow   = errors.New("ingest: input sequence exhausted")
	ErrRetryExhausted     = errors.New("ingest: transaction retry budget exhausted")
	ErrCommitUnknown      = errors.New("ingest: commit outcome is unknown; durable reconciliation required")
	ErrStorageUnavailable = errors.New("ingest: durable storage is unavailable")
	ErrQueueFull          = errors.New("ingest: bounded ingestion queue is full")
	ErrBatcherClosed      = errors.New("ingest: batcher is closed")
)
