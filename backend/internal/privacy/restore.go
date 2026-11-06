package privacy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/reducer"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
	"github.com/jackc/pgx/v5"
)

var (
	ErrRestoreIncomplete = errors.New("privacy: restored deletion replay incomplete")
	ErrRestoreTarget     = errors.New("privacy: invalid restore target identity")
)

var restoreTargetPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type RestoreResult struct {
	LedgerID        string
	LastSeq         uint64
	HeadHash        string
	Applied         int
	Skipped         int
	RunID           string
	AppliedErasures []string
	Proof           CoverageProof
	Ready           bool
}

type RestoreCoordinator struct {
	Ledger     *Ledger
	Pool       *storage.Pool
	NewManager func(villageID string) (*writer.Manager, error)
	NewReducer func(*writer.Manager) (*reducer.Reducer, error)
}

// Reapply independently discovers deletion payloads from one immutable verified ledger
// snapshot and applies missing anonymization boundaries to a restored database. The
// caller does not supply actor identities.
func (c *RestoreCoordinator) Reapply(ctx context.Context, targetIdentity string) (RestoreResult, error) {
	if !restoreTargetPattern.MatchString(targetIdentity) {
		return RestoreResult{}, ErrRestoreTarget
	}
	if c == nil || c.Ledger == nil || c.Pool == nil || c.NewManager == nil || c.NewReducer == nil {
		return RestoreResult{}, ErrRestoreIncomplete
	}
	snapshot, err := c.Ledger.VerifySnapshot(targetIdentity)
	if err != nil {
		return RestoreResult{}, err
	}
	if snapshot.Proof.LastSeq == 0 {
		return RestoreResult{}, ErrLedgerCoverage
	}
	managers := map[string]*writer.Manager{}
	applied := make([]string, 0)
	appliedCount, skippedCount := 0, 0
	for _, deletion := range snapshot.Deletions {
		payload := deletion.Payload
		var villageID string
		err := func() error {
			tx, err := c.Pool.Begin(ctx)
			if err != nil {
				return err
			}
			defer tx.Rollback(ctx)
			return tx.QueryRow(ctx, `SELECT village_id::text FROM ttg.sessions WHERE id = $1`, payload.SessionID).Scan(&villageID)
		}()
		if errors.Is(err, pgx.ErrNoRows) {
			return RestoreResult{}, ErrRestoreIncomplete
		}
		if err != nil {
			return RestoreResult{}, err
		}
		present, err := c.subjectPresent(ctx, villageID, payload.SessionID, payload.ActorRefProtected)
		if err != nil {
			return RestoreResult{}, err
		}
		if !present {
			skippedCount++
			continue
		}
		if err := c.materializeTombstone(ctx, payload); err != nil {
			return RestoreResult{}, err
		}
		manager := managers[villageID]
		if manager == nil {
			manager, err = c.NewManager(villageID)
			if err != nil {
				return RestoreResult{}, err
			}
			managers[villageID] = manager
		}
		red, err := c.NewReducer(manager)
		if err != nil {
			return RestoreResult{}, err
		}
		lease, err := acquireWithRetry(ctx, manager, villageID, 15*time.Second)
		if err != nil {
			return RestoreResult{}, err
		}
		result, err := red.AnonymizeWithCompletion(ctx, lease, payload.SessionID, []string{payload.ActorRefProtected}, func(ctx context.Context, tx writer.FencedTx, revision int64) error {
			var runID string
			if err := tx.QueryRow(ctx, `INSERT INTO ttg.privacy_restore_runs (id, ledger_id, ledger_last_seq, ledger_head_hash, target_identity, result_hash)
				VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
				ON CONFLICT (ledger_id, ledger_last_seq, target_identity) DO UPDATE SET target_identity = EXCLUDED.target_identity
				RETURNING id::text`, snapshot.Proof.LedgerID, snapshot.Proof.LastSeq, snapshot.Proof.HeadHash, targetIdentity, hashString(snapshot.Proof.HeadHash+payload.TombstoneID)).Scan(&runID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO ttg.privacy_restore_erasures (run_id, tombstone_id, session_id, actor_ref_protected, applied_revision)
				VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, runID, payload.TombstoneID, payload.SessionID, payload.ActorRefProtected, revision); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `UPDATE ttg.privacy_tombstones SET restore_applied_at = now() WHERE id = $1`, payload.TombstoneID)
			return err
		})
		if err != nil {
			return RestoreResult{}, err
		}
		applied = append(applied, payload.TombstoneID)
		appliedCount++
		_ = result
	}
	ready := appliedCount+skippedCount == len(snapshot.Deletions)
	return RestoreResult{
		LedgerID: snapshot.Proof.LedgerID, LastSeq: snapshot.Proof.LastSeq, HeadHash: snapshot.Proof.HeadHash,
		Applied: appliedCount, Skipped: skippedCount, AppliedErasures: applied, Proof: snapshot.Proof, Ready: ready,
	}, nil
}

func (c *RestoreCoordinator) subjectPresent(ctx context.Context, villageID, sessionID, actor string) (bool, error) {
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var present bool
	err = tx.QueryRow(ctx, `SELECT
		EXISTS (SELECT 1 FROM ttg.players WHERE village_id = $1 AND actor_key = $3)
		OR EXISTS (SELECT 1 FROM ttg.actor_identity_aliases WHERE village_id = $1 AND actor_key = $3)
		OR EXISTS (SELECT 1 FROM ttg.wave_members WHERE session_id = $2 AND actor_key = $3)
		OR EXISTS (SELECT 1 FROM ttg.session_player_progress WHERE session_id = $2 AND actor_key = $3)
		OR EXISTS (SELECT 1 FROM ttg.input_events WHERE session_id = $2 AND actor_key = $3)
		OR EXISTS (SELECT 1 FROM ttg.ballots WHERE session_id = $2 AND actor_key = $3)`, villageID, sessionID, actor).Scan(&present)
	if err != nil {
		return false, err
	}
	return present, nil
}

func (c *RestoreCoordinator) materializeTombstone(ctx context.Context, payload LedgerPayload) error {
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.privacy_tombstones (id, actor_ref_protected, scope) VALUES ($1::uuid, $2, $3) ON CONFLICT (id) DO NOTHING`, payload.TombstoneID, payload.ActorRefProtected, payload.Scope); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func acquireWithRetry(ctx context.Context, manager *writer.Manager, villageID string, timeout time.Duration) (writer.Lease, error) {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		lease, err := manager.Acquire(ctx, villageID)
		if err == nil {
			return lease, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return writer.Lease{}, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return writer.Lease{}, last
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
