package writer

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	ensureLeaseSQL = `
INSERT INTO ttg.writer_leases (village_id, writer_epoch, owner_instance, leased_until)
SELECT $1::uuid, 0, NULL, NULL
FROM ttg.villages
WHERE id = $1::uuid
ON CONFLICT (village_id) DO NOTHING`

	lockLeaseSQL = `
SELECT writer_epoch, owner_instance, leased_until
FROM ttg.writer_leases
WHERE village_id = $1::uuid
FOR UPDATE`

	acquireLeaseSQL = `
UPDATE ttg.writer_leases
SET writer_epoch = $2,
    owner_instance = $3,
    leased_until = clock_timestamp() + ($4::bigint * interval '1 millisecond')
WHERE village_id = $1::uuid
RETURNING leased_until`

	renewLeaseSQL = `
UPDATE ttg.writer_leases
SET leased_until = clock_timestamp() + ($4::bigint * interval '1 millisecond')
WHERE village_id = $1::uuid AND owner_instance = $2 AND writer_epoch = $3
RETURNING leased_until`

	releaseLeaseSQL = `
UPDATE ttg.writer_leases
SET owner_instance = NULL, leased_until = NULL
WHERE village_id = $1::uuid AND owner_instance = $2 AND writer_epoch = $3`

	selectSessionForUpdateSQL = `
SELECT id::text, village_id::text, writer_epoch, lifecycle
FROM ttg.sessions
WHERE village_id = $1::uuid AND lifecycle <> 'stopped'
FOR UPDATE`

	selectSessionByIDForUpdateSQL = `
SELECT id::text, village_id::text, writer_epoch, lifecycle
FROM ttg.sessions
WHERE id = $1::uuid
FOR UPDATE`

	syncSessionEpochSQL = `
UPDATE ttg.sessions
SET writer_epoch = $2, last_heartbeat_at = clock_timestamp()
WHERE id = $1::uuid`

	clearSessionEpochSQL = `
UPDATE ttg.sessions
SET writer_epoch = 0, last_heartbeat_at = clock_timestamp()
WHERE id = $1::uuid`
)

// Manager owns one local writer identity. It never claims ownership from process
// memory; every operation that matters verifies committed PostgreSQL state.
type Manager struct {
	pool     *storage.Pool
	instance string
	config   Config

	mu       sync.Mutex
	lease    Lease
	verified bool
	lastErr  string
	epoch    atomic.Int64

	// Test-only hooks. Production leaves them nil.
	beforeCommit func() error
	commit       func(context.Context, pgx.Tx) error
}

// New validates the manager configuration. It performs no database I/O.
func New(pool *storage.Pool, instance string, config Config) (*Manager, error) {
	if pool == nil {
		return nil, ErrStorageUnavailable
	}
	if err := validateInstance(instance); err != nil {
		return nil, err
	}
	normalized, err := config.normalized()
	if err != nil {
		return nil, err
	}
	return &Manager{pool: pool, instance: instance, config: normalized}, nil
}

// Acquire atomically acquires or renews the village lease. A valid lease held by a
// different instance is rejected. A same-owner valid lease keeps its epoch and only
// extends expiry. An expired/absent ownership generation receives epoch+1.
func (m *Manager) Acquire(ctx context.Context, villageID string) (Lease, error) {
	if err := validateVillage(villageID); err != nil {
		m.clear(err)
		return Lease{}, err
	}
	m.mu.Lock()
	busy := m.verified && m.lease.VillageID != "" && m.lease.VillageID != villageID && time.Now().Before(m.lease.LeasedUntil)
	m.mu.Unlock()
	if busy {
		return Lease{}, ErrManagerBusy
	}
	var result Lease
	err := m.runTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, ensureLeaseSQL, villageID); err != nil {
			return err
		}
		current, now, err := lockLease(ctx, tx, villageID)
		if err != nil {
			return err
		}
		valid := current.Until.Valid && current.Until.Time.After(now)
		sameOwner := current.Owner.Valid && current.Owner.String == m.instance
		if valid && sameOwner {
			var until time.Time
			if err := tx.QueryRow(ctx, renewLeaseSQL, villageID, m.instance, current.Epoch, m.config.LeaseDuration.Milliseconds()).Scan(&until); err != nil {
				return err
			}
			sessionID, err := syncActiveSession(ctx, tx, villageID, current.Epoch)
			if err != nil {
				return err
			}
			result = Lease{VillageID: villageID, SessionID: sessionID, Epoch: current.Epoch, OwnerInstance: m.instance, LeasedUntil: until}
			return nil
		}
		if valid {
			return ErrLeaseHeld
		}
		if current.Epoch < 0 {
			return ErrInvalidEpoch
		}
		if current.Epoch == math.MaxInt64 {
			return ErrEpochExhausted
		}
		next := current.Epoch + 1
		if next <= 0 {
			return ErrEpochExhausted
		}
		var until time.Time
		if err := tx.QueryRow(ctx, acquireLeaseSQL, villageID, next, m.instance, m.config.LeaseDuration.Milliseconds()).Scan(&until); err != nil {
			return err
		}
		sessionID, err := syncActiveSession(ctx, tx, villageID, next)
		if err != nil {
			return err
		}
		result = Lease{VillageID: villageID, SessionID: sessionID, Epoch: next, OwnerInstance: m.instance, LeasedUntil: until}
		return nil
	})
	if err != nil {
		m.clear(err)
		return Lease{}, err
	}
	m.setLease(result)
	return result, nil
}

// Renew extends a still-valid lease without changing its epoch. Expired leases are
// never revived by renewal.
func (m *Manager) Renew(ctx context.Context, lease Lease) (Lease, error) {
	if err := m.validateLease(lease); err != nil {
		m.clear(err)
		return Lease{}, err
	}
	var result Lease
	err := m.runTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		current, now, err := lockLease(ctx, tx, lease.VillageID)
		if err != nil {
			return err
		}
		if current.Epoch != lease.Epoch || !current.Owner.Valid || current.Owner.String != m.instance {
			return ErrLeaseLost
		}
		if !current.Until.Valid || !current.Until.Time.After(now) {
			return ErrLeaseExpired
		}
		var until time.Time
		if err := tx.QueryRow(ctx, renewLeaseSQL, lease.VillageID, m.instance, lease.Epoch, m.config.LeaseDuration.Milliseconds()).Scan(&until); err != nil {
			return err
		}
		sessionID, err := syncActiveSession(ctx, tx, lease.VillageID, lease.Epoch)
		if err != nil {
			return err
		}
		result = Lease{VillageID: lease.VillageID, SessionID: sessionID, Epoch: lease.Epoch, OwnerInstance: m.instance, LeasedUntil: until}
		return nil
	})
	if err != nil {
		m.clear(err)
		return Lease{}, err
	}
	m.setLease(result)
	return result, nil
}

// Release conditionally clears a current, unexpired lease owned by this instance and
// epoch. It never decrements the lease epoch.
func (m *Manager) Release(ctx context.Context, lease Lease) error {
	if err := m.validateLease(lease); err != nil {
		m.clear(err)
		return err
	}
	err := m.runTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		current, now, err := lockLease(ctx, tx, lease.VillageID)
		if err != nil {
			return err
		}
		if current.Epoch != lease.Epoch || !current.Owner.Valid || current.Owner.String != m.instance {
			return ErrLeaseLost
		}
		if !current.Until.Valid || !current.Until.Time.After(now) {
			return ErrLeaseExpired
		}
		if _, err := tx.Exec(ctx, releaseLeaseSQL, lease.VillageID, m.instance, lease.Epoch); err != nil {
			return err
		}
		return clearActiveSession(ctx, tx, lease.VillageID, lease.Epoch)
	})
	m.clear(err)
	return err
}

// WithFence runs fn inside a transaction that has verified the current committed
// village lease, owner, epoch, expiry and session binding. The lease row is locked
// before the check and held until commit, so a takeover cannot interleave between the
// check and the protected write. The callback cannot commit or roll back independently.
func (m *Manager) WithFence(ctx context.Context, lease Lease, sessionID string, fn func(context.Context, FencedTx) error) error {
	if err := m.validateLease(lease); err != nil {
		m.clear(err)
		return err
	}
	if err := validateSession(sessionID); err != nil {
		m.clear(err)
		return err
	}
	if lease.OwnerInstance != m.instance {
		m.clear(ErrLeaseLost)
		return ErrLeaseLost
	}
	if fn == nil {
		return ErrInvalidSession
	}
	err := m.runTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		current, now, err := lockLease(ctx, tx, lease.VillageID)
		if err != nil {
			return err
		}
		if current.Epoch != lease.Epoch || !current.Owner.Valid || current.Owner.String != m.instance {
			return ErrLeaseLost
		}
		if !current.Until.Valid || !current.Until.Time.After(now) {
			return ErrLeaseExpired
		}
		var sessionVillage, lifecycle string
		var sessionEpoch int64
		err = tx.QueryRow(ctx, selectSessionByIDForUpdateSQL, sessionID).
			Scan(&sessionID, &sessionVillage, &sessionEpoch, &lifecycle)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionNotFound
		}
		if err != nil {
			return err
		}
		if sessionVillage != lease.VillageID {
			return ErrSessionMismatch
		}
		if lifecycle == "stopped" {
			return ErrSessionInactive
		}
		if sessionEpoch != lease.Epoch {
			return ErrStaleEpoch
		}
		if err := fn(ctx, fencedTx{tx: tx}); err != nil {
			return callbackError{err: err}
		}
		return nil
	})
	if err != nil {
		m.clear(err)
	}
	return err
}

// Close releases the current lease with the caller's bounded context. It never waits
// unbounded and never leaves a verified local ownership claim after return.
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	lease := m.lease
	verified := m.verified
	m.mu.Unlock()
	if !verified {
		m.clear(nil)
		return nil
	}
	err := m.Release(ctx, lease)
	m.clear(err)
	return err
}

// Epoch returns the last verified local epoch, or 0. A locally elapsed lease is
// fail-closed to 0 for readiness; this can only remove authority, never grant it.
// Callers must still use WithFence for authoritative writes.
func (m *Manager) Epoch() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.localVerifiedLocked(time.Now()) {
		return 0
	}
	return m.epoch.Load()
}

func (m *Manager) localVerifiedLocked(now time.Time) bool {
	return m.verified && !m.lease.LeasedUntil.IsZero() && now.Before(m.lease.LeasedUntil)
}

// Status reports the truthful local view for readiness/metrics integration.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	verified := m.localVerifiedLocked(time.Now())
	epoch := int64(0)
	if verified {
		epoch = m.epoch.Load()
	}
	return Status{
		Verified:  verified,
		Epoch:     epoch,
		VillageID: m.lease.VillageID,
		SessionID: m.lease.SessionID,
		ExpiresAt: m.lease.LeasedUntil,
		LastError: m.lastErr,
	}
}

func (m *Manager) validateLease(lease Lease) error {
	if err := validateVillage(lease.VillageID); err != nil {
		return err
	}
	if lease.OwnerInstance != m.instance {
		return ErrLeaseLost
	}
	if lease.Epoch <= 0 {
		return ErrInvalidEpoch
	}
	return nil
}

func (m *Manager) setLease(lease Lease) {
	m.mu.Lock()
	m.lease = lease
	m.verified = true
	m.lastErr = ""
	m.mu.Unlock()
	m.epoch.Store(lease.Epoch)
}

func (m *Manager) clear(err error) {
	m.mu.Lock()
	m.lease = Lease{}
	m.verified = false
	m.lastErr = Code(err)
	m.mu.Unlock()
	m.epoch.Store(0)
}

func (m *Manager) runTx(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	for attempt := 1; attempt <= 3; attempt++ {
		tx, err := m.pool.Begin(ctx)
		if err != nil {
			return ErrStorageUnavailable
		}
		err = fn(ctx, tx)
		if err == nil && m.beforeCommit != nil {
			err = m.beforeCommit()
		}
		if err == nil {
			commit := m.commit
			if commit == nil {
				commit = func(ctx context.Context, tx pgx.Tx) error { return tx.Commit(ctx) }
			}
			if err := commit(ctx, tx); err != nil {
				_ = tx.Rollback(ctx)
				return ErrCommitUnknown
			}
			return nil
		}
		_ = tx.Rollback(ctx)
		if isRetryable(err) {
			if attempt < 3 {
				continue
			}
			return ErrRetryExhausted
		}
		return classify(err)
	}
	return ErrRetryExhausted
}

type leaseRow struct {
	Epoch int64
	Owner pgtype.Text
	Until pgtype.Timestamptz
}

func lockLease(ctx context.Context, tx pgx.Tx, villageID string) (leaseRow, time.Time, error) {
	var row leaseRow
	err := tx.QueryRow(ctx, lockLeaseSQL, villageID).Scan(&row.Epoch, &row.Owner, &row.Until)
	if errors.Is(err, pgx.ErrNoRows) {
		return leaseRow{}, time.Time{}, ErrVillageNotFound
	}
	if err != nil {
		return leaseRow{}, time.Time{}, err
	}
	var now time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return leaseRow{}, time.Time{}, err
	}
	return row, now, nil
}

func syncActiveSession(ctx context.Context, tx pgx.Tx, villageID string, epoch int64) (string, error) {
	var sessionID, sessionVillage, lifecycle string
	var sessionEpoch int64
	err := tx.QueryRow(ctx, selectSessionForUpdateSQL, villageID).
		Scan(&sessionID, &sessionVillage, &sessionEpoch, &lifecycle)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if sessionEpoch > epoch {
		return "", ErrInconsistentEpoch
	}
	if _, err := tx.Exec(ctx, syncSessionEpochSQL, sessionID, epoch); err != nil {
		return "", err
	}
	return sessionID, nil
}

func clearActiveSession(ctx context.Context, tx pgx.Tx, villageID string, epoch int64) error {
	var sessionID, sessionVillage, lifecycle string
	var sessionEpoch int64
	err := tx.QueryRow(ctx, selectSessionForUpdateSQL, villageID).
		Scan(&sessionID, &sessionVillage, &sessionEpoch, &lifecycle)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if sessionEpoch > epoch {
		return ErrInconsistentEpoch
	}
	_, err = tx.Exec(ctx, clearSessionEpochSQL, sessionID)
	return err
}

type fencedTx struct {
	tx pgx.Tx
}

func (f fencedTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return f.tx.Exec(ctx, sql, args...)
}

func (f fencedTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return f.tx.Query(ctx, sql, args...)
}

func (f fencedTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return f.tx.QueryRow(ctx, sql, args...)
}

func validateVillage(id string) error {
	var parsed pgtype.UUID
	if err := parsed.Scan(id); err != nil || !parsed.Valid {
		return ErrInvalidVillage
	}
	return nil
}

func validateSession(id string) error {
	var parsed pgtype.UUID
	if err := parsed.Scan(id); err != nil || !parsed.Valid {
		return ErrInvalidSession
	}
	return nil
}

func validateInstance(instance string) error {
	if instance == "" || len(instance) > MaxInstanceBytes || !utf8.ValidString(instance) {
		return ErrInvalidInstance
	}
	if strings.TrimSpace(instance) != instance {
		return ErrInvalidInstance
	}
	for _, r := range instance {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ErrInvalidInstance
		}
	}
	return nil
}

type callbackError struct{ err error }

func (e callbackError) Error() string { return "writer: fenced callback failed" }
func (e callbackError) Unwrap() error { return e.err }

func isRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "40001" || pgErr.Code == "40P01"
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	var callback callbackError
	if errors.As(err, &callback) {
		return err
	}
	for _, sentinel := range []error{
		ErrInvalidVillage, ErrInvalidSession, ErrInvalidInstance, ErrInvalidDuration,
		ErrVillageNotFound, ErrSessionNotFound, ErrSessionMismatch, ErrSessionInactive,
		ErrLeaseHeld, ErrManagerBusy, ErrLeaseExpired, ErrLeaseLost, ErrStaleEpoch, ErrInconsistentEpoch,
		ErrEpochExhausted, ErrInvalidEpoch, ErrStorageUnavailable, ErrRetryExhausted,
		ErrCommitUnknown,
	} {
		if errors.Is(err, sentinel) {
			return err
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40001", "40P01":
			return ErrRetryExhausted
		case "23503":
			return ErrVillageNotFound
		case "22P02":
			return ErrInvalidVillage
		}
	}
	return ErrStorageUnavailable
}
