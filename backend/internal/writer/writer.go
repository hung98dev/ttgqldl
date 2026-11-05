// Package writer implements the PostgreSQL-authoritative simulation writer lease.
//
// PostgreSQL is the sole authority: acquisition, renewal, release and every fenced
// mutation lock ttg.writer_leases and compare the committed epoch/owner/expiry using
// PostgreSQL's clock_timestamp(). A fenced transaction holds the lease row lock until
// commit or rollback, so a takeover cannot commit between the authority check and the
// protected write. The package deliberately does not implement the W0402 reducer,
// outbox, checkpoints or crash recovery.
package writer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Lease duration defaults and bounds.
const (
	DefaultLeaseDuration = 5 * time.Second
	MinLeaseDuration     = 200 * time.Millisecond
	MaxLeaseDuration     = time.Minute
	MaxInstanceBytes     = 128
)

// Sentinel errors are privacy-safe: they never contain village/session IDs, owner
// strings, SQL text or raw provider data.
var (
	ErrInvalidVillage     = errors.New("writer: invalid village")
	ErrInvalidSession     = errors.New("writer: invalid session")
	ErrInvalidInstance    = errors.New("writer: invalid instance")
	ErrInvalidDuration    = errors.New("writer: invalid lease duration")
	ErrVillageNotFound    = errors.New("writer: village not found")
	ErrSessionNotFound    = errors.New("writer: session not found")
	ErrSessionMismatch    = errors.New("writer: session does not belong to village")
	ErrSessionInactive    = errors.New("writer: session is stopped")
	ErrLeaseHeld          = errors.New("writer: lease is held by another writer")
	ErrManagerBusy        = errors.New("writer: manager already owns another village")
	ErrLeaseExpired       = errors.New("writer: lease has expired")
	ErrLeaseLost          = errors.New("writer: lease ownership is no longer current")
	ErrStaleEpoch         = errors.New("writer: stale fencing epoch")
	ErrInconsistentEpoch  = errors.New("writer: session and lease epochs disagree")
	ErrEpochExhausted     = errors.New("writer: fencing epoch exhausted")
	ErrInvalidEpoch       = errors.New("writer: invalid fencing epoch")
	ErrStorageUnavailable = errors.New("writer: storage unavailable")
	ErrRetryExhausted     = errors.New("writer: transaction retry exhausted")
	ErrCommitUnknown      = errors.New("writer: commit outcome is unknown")
)

// Config controls lease timing. There is no background heartbeat worker in W0401;
// callers renew explicitly. The interval is deliberately not guessed here.
type Config struct {
	LeaseDuration time.Duration
}

func (c Config) normalized() (Config, error) {
	if c.LeaseDuration == 0 {
		c.LeaseDuration = DefaultLeaseDuration
	}
	if c.LeaseDuration < MinLeaseDuration || c.LeaseDuration > MaxLeaseDuration {
		return Config{}, ErrInvalidDuration
	}
	return c, nil
}

// Lease is the caller-visible ownership record returned only after a committed
// acquisition or renewal. SessionID may be empty when the village has no active
// session.
type Lease struct {
	VillageID     string
	SessionID     string
	Epoch         int64
	OwnerInstance string
	LeasedUntil   time.Time
}

// Status is the truthful local ownership view. Verified is false whenever the manager
// has not confirmed current ownership through PostgreSQL.
type Status struct {
	Verified  bool
	Epoch     int64
	VillageID string
	SessionID string
	ExpiresAt time.Time
	LastError string
}

// Code returns a fixed, privacy-safe code for a writer error.
func Code(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrLeaseHeld):
		return "lease_held"
	case errors.Is(err, ErrManagerBusy):
		return "manager_busy"
	case errors.Is(err, ErrLeaseExpired):
		return "lease_expired"
	case errors.Is(err, ErrLeaseLost):
		return "lease_lost"
	case errors.Is(err, ErrStaleEpoch):
		return "stale_epoch"
	case errors.Is(err, ErrEpochExhausted):
		return "epoch_exhausted"
	case errors.Is(err, ErrInconsistentEpoch):
		return "inconsistent_epoch"
	case errors.Is(err, ErrSessionMismatch):
		return "session_mismatch"
	case errors.Is(err, ErrSessionInactive):
		return "session_inactive"
	case errors.Is(err, ErrSessionNotFound):
		return "session_not_found"
	case errors.Is(err, ErrVillageNotFound):
		return "village_not_found"
	case errors.Is(err, ErrStorageUnavailable):
		return "storage_unavailable"
	case errors.Is(err, ErrRetryExhausted):
		return "retry_exhausted"
	case errors.Is(err, ErrCommitUnknown):
		return "commit_unknown"
	default:
		return "invalid_input"
	}
}

// FencedTx is the only transaction surface exposed to a fenced callback. It has no
// Commit or Rollback method: the Manager owns durability and always rolls back on a
// callback error.
type FencedTx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (l Lease) String() string {
	return fmt.Sprintf("epoch=%d", l.Epoch)
}
