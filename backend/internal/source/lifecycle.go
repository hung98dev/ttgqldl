package source

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// lifecycle implements the shared adapter state machine: exactly one active run,
// synchronous delivery on the caller's goroutine, prompt cancellation for a blocked
// sink, and cursor accounting that only advances after a successful delivery.
//
// No goroutine is ever spawned here. Start and Reconnect run on the caller's
// goroutine, which makes lifecycle and leak behaviour deterministic.
type lifecycle struct {
	mu        sync.Mutex
	status    Status
	running   bool
	stopCh    chan struct{}
	stopOnce  *sync.Once
	doneCh    chan struct{}
	cancel    context.CancelFunc
	delivered uint64
	cursor    Cursor
	lastError string
}

func (l *lifecycle) health() Health {
	l.mu.Lock()
	defer l.mu.Unlock()
	return Health{
		Status:    l.status,
		Delivered: l.delivered,
		Cursor:    l.cursor,
		LastError: l.lastError,
	}
}

// begin starts a run. reconnect selects the stricter entry state: a fresh Start is
// only legal from StatusNew, while Reconnect is only legal after EOF, disconnection
// or an operator stop.
func (l *lifecycle) begin(ctx context.Context, cancel context.CancelFunc, session Session, sourceID string, next uint64, reconnect bool) error {
	if !session.Valid() {
		return ErrInvalidSession
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.running {
		return ErrAlreadyStarted
	}
	if reconnect {
		switch l.status {
		case StatusEOF, StatusDisconnected, StatusStopped:
		default:
			return fmt.Errorf("%w: status %s", ErrNotReconnectable, l.status)
		}
	} else if l.status != StatusNew {
		return fmt.Errorf("%w: status %s", ErrAlreadyStarted, l.status)
	}
	l.status = StatusRunning
	l.running = true
	l.stopCh = make(chan struct{})
	l.stopOnce = &sync.Once{}
	l.doneCh = make(chan struct{})
	l.cancel = cancel
	l.cursor = NewCursor(session.ID, sourceID, next)
	l.lastError = ""
	return nil
}

func (l *lifecycle) stopping() bool {
	l.mu.Lock()
	stopCh := l.stopCh
	l.mu.Unlock()
	if stopCh == nil {
		return false
	}
	select {
	case <-stopCh:
		return true
	default:
		return false
	}
}

// deliver invokes the sink and advances the cursor only on success. A callback
// failure never moves the cursor, so a reconnect redelivers the rejected event.
func (l *lifecycle) deliver(ctx context.Context, sink Sink, event Event, next uint64) error {
	if l.stopping() {
		return ErrSourceStopped
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := sink(ctx, event); err != nil {
		if l.stopping() {
			return ErrSourceStopped
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("%w: callback failed", ErrDeliveryRejected)
	}
	l.mu.Lock()
	l.delivered++
	l.cursor.NextEvent = next
	l.mu.Unlock()
	return nil
}

// finish records the terminal status and releases Stop waiters. It runs exactly once
// per successful begin, from the delivery goroutine.
func (l *lifecycle) finish(status Status, runErr error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.running = false
	l.status = status
	l.lastError = errorCategory(runErr)
	l.cancel = nil
	close(l.doneCh)
}

// stop closes the stop signal, cancels the run context (so a blocked sink can
// return), and waits for the delivery loop to exit. After a successful stop no
// callback is running or can start.
func (l *lifecycle) stop(ctx context.Context) error {
	l.mu.Lock()
	if !l.running {
		if l.status == StatusNew {
			l.status = StatusStopped
		}
		l.mu.Unlock()
		return nil
	}
	stopOnce := l.stopOnce
	stopCh := l.stopCh
	doneCh := l.doneCh
	cancel := l.cancel
	l.mu.Unlock()

	stopOnce.Do(func() { close(stopCh) })
	if cancel != nil {
		cancel()
	}
	select {
	case <-doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// errorCategory maps an error to a stable health category. It never returns raw
// error text, so identifiers, comments and aliases cannot leak through Health.
func errorCategory(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, ErrDeliveryRejected):
		return "delivery_rejected"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "cancelled"
	case errors.Is(err, ErrSourceStopped):
		return "stopped"
	case errors.Is(err, ErrSessionMismatch):
		return "session_mismatch"
	case errors.Is(err, ErrProviderMismatch):
		return "provider_mismatch"
	case errors.Is(err, ErrInvalidCursor):
		return "invalid_cursor"
	case errors.Is(err, ErrCursorMismatch):
		return "cursor_mismatch"
	case errors.Is(err, ErrRecordTooLong):
		return "record_too_long"
	case errors.Is(err, ErrSchemaUnsupported):
		return "schema_unsupported"
	case errors.Is(err, ErrFixtureDigest):
		return "fixture_digest"
	case errors.Is(err, ErrMalformedRecord), errors.Is(err, ErrRecordEncoding):
		return "malformed_record"
	case errors.Is(err, ErrInvalidEvent), errors.Is(err, ErrInvalidMessage), errors.Is(err, ErrInvalidAlias), errors.Is(err, ErrInvalidMetadata):
		return "invalid_event"
	default:
		return "failed"
	}
}
