// Package recovery coordinates a fresh writer process with the W0401 lease and the
// W0403 checkpoint/replay verifier. It never restores from process memory and never
// writes authoritative state during verification.
package recovery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/reducer"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
)

type State string

const (
	StateUninitialized State = "UNINITIALIZED"
	StateAcquiring     State = "ACQUIRING_LEASE"
	StateVerifying     State = "VERIFYING_HISTORY"
	StateReady         State = "READY"
	StateIntegrityFail State = "INTEGRITY_FAILED"
	StateStopped       State = "STOPPED"
)

var (
	ErrIntegrity = errors.New("recovery: authoritative state failed integrity verification")
	ErrStale     = errors.New("recovery: writer lease is no longer current")
)

type Result struct {
	Lease     writer.Lease
	Revision  int64
	Tick      int
	StateHash string
	Pristine  bool
	Elapsed   time.Duration
}

type Coordinator struct {
	pool      *storage.Pool
	manager   *writer.Manager
	reducer   *reducer.Reducer
	sessionID string
	villageID string

	mu    sync.Mutex
	state State
	lease writer.Lease
}

func New(pool *storage.Pool, manager *writer.Manager, red *reducer.Reducer, sessionID, villageID string) (*Coordinator, error) {
	if pool == nil || manager == nil || red == nil || sessionID == "" || villageID == "" {
		return nil, errors.New("recovery: pool, manager, reducer and identities are required")
	}
	return &Coordinator{pool: pool, manager: manager, reducer: red, sessionID: sessionID, villageID: villageID, state: StateUninitialized}, nil
}

// Recover acquires a fresh epoch, verifies the committed session/checkpoint/replay
// history while holding the lease, and only then enters READY.
func (c *Coordinator) Recover(ctx context.Context) (Result, error) {
	started := time.Now()
	c.setState(StateAcquiring)
	lease, err := c.manager.Acquire(ctx, c.villageID)
	if err != nil {
		c.setState(StateIntegrityFail)
		return Result{}, err
	}
	c.mu.Lock()
	c.lease = lease
	c.mu.Unlock()
	c.setState(StateVerifying)

	revision, tick, hash, runtimePresent, err := c.loadSession(ctx)
	if err != nil {
		c.fail(ctx, err)
		return Result{}, err
	}
	if revision == 0 && !runtimePresent && tick == -1 {
		if _, err := c.manager.Renew(ctx, lease); err != nil {
			c.fail(ctx, err)
			return Result{}, err
		}
		result := Result{Lease: lease, Revision: 0, Tick: -1, Pristine: true, Elapsed: time.Since(started)}
		c.setState(StateReady)
		return result, nil
	}
	replay, err := c.reducer.VerifyReplay(ctx, c.sessionID, 0)
	if err != nil {
		c.fail(ctx, fmt.Errorf("%w: %v", ErrIntegrity, err))
		return Result{}, ErrIntegrity
	}
	renewed, err := c.manager.Renew(ctx, lease)
	if err != nil {
		c.fail(ctx, err)
		return Result{}, err
	}
	revision2, tick2, hash2, present2, err := c.loadSession(ctx)
	if err != nil {
		c.fail(ctx, err)
		return Result{}, err
	}
	if !present2 || revision2 != revision || tick2 != tick || hash2 != hash ||
		replay.ToRevision != revision || replay.StateHash != hash {
		c.fail(ctx, ErrStale)
		return Result{}, ErrStale
	}
	c.mu.Lock()
	c.lease = renewed
	c.mu.Unlock()
	c.setState(StateReady)
	return Result{Lease: renewed, Revision: revision, Tick: int(tick), StateHash: hash, Elapsed: time.Since(started)}, nil
}

// Step runs one fenced reducer transition after READY.
func (c *Coordinator) Step(ctx context.Context) (reducer.Result, error) {
	c.mu.Lock()
	state, lease := c.state, c.lease
	c.mu.Unlock()
	if state != StateReady {
		return reducer.Result{}, fmt.Errorf("recovery: state %s is not ready", state)
	}
	return c.reducer.Step(ctx, lease, c.sessionID)
}

func (c *Coordinator) Close(ctx context.Context) error {
	err := c.manager.Close(ctx)
	c.setState(StateStopped)
	return err
}

func (c *Coordinator) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *Coordinator) setState(state State) {
	c.mu.Lock()
	c.state = state
	c.mu.Unlock()
}

func (c *Coordinator) fail(ctx context.Context, err error) {
	c.setState(StateIntegrityFail)
	_ = c.manager.Close(ctx)
	_ = err
}

func (c *Coordinator) loadSession(ctx context.Context) (revision int64, tick int32, hash string, runtimePresent bool, err error) {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return 0, 0, "", false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var villageID, rules string
	var runtime []byte
	var hashPtr *string
	err = tx.QueryRow(ctx, `SELECT village_id::text, rules_version, reducer_revision, wave_tick,
		runtime_state, last_state_hash FROM ttg.sessions WHERE id = $1`, c.sessionID).Scan(
		&villageID, &rules, &revision, &tick, &runtime, &hashPtr,
	)
	if err != nil {
		return 0, 0, "", false, err
	}
	if villageID != c.villageID {
		return 0, 0, "", false, ErrIntegrity
	}
	if rules != domain.RulesVersion {
		return 0, 0, "", false, ErrIntegrity
	}
	if hashPtr != nil {
		hash = *hashPtr
	}
	return revision, tick, hash, len(runtime) > 0, nil
}
