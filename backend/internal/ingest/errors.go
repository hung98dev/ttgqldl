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
)
