package privacy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/reducer"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
)

var ErrRestoreIncomplete = errors.New("privacy: restored deletion replay incomplete")

type RestoreResult struct {
	LedgerID        string
	LastSeq         uint64
	HeadHash        string
	Applied         int
	Skipped         int
	RunID           string
	AppliedErasures []string
}

type RestoreCoordinator struct {
	Ledger     *Ledger
	Pool       *storage.Pool
	NewManager func(villageID string) (*writer.Manager, error)
	NewReducer func(*writer.Manager) (*reducer.Reducer, error)
}

// Reapply independently discovers deletion payloads from the verified encrypted ledger
// and applies missing anonymization boundaries to a restored database. The caller does
// not supply actor identities.
func (c *RestoreCoordinator) Reapply(ctx context.Context, targetIdentity string) (RestoreResult, error) {
	if c == nil || c.Ledger == nil || c.Pool == nil || c.NewManager == nil || c.NewReducer == nil {
		return RestoreResult{}, ErrRestoreIncomplete
	}
	proof, err := c.Ledger.Verify(targetIdentity)
	if err != nil {
		return RestoreResult{}, err
	}
	if proof.LastSeq == 0 {
		return RestoreResult{}, ErrLedgerCoverage
	}
	deletions, err := c.Ledger.Deletions()
	if err != nil {
		return RestoreResult{}, err
	}
	managers := map[string]*writer.Manager{}
	applied := make([]string, 0)
	appliedCount, skippedCount := 0, 0
	for _, deletion := range deletions {
		payload := deletion.Payload
		var villageID string
		var epoch int64
		err := func() error {
			tx, err := c.Pool.Begin(ctx)
			if err != nil {
				return err
			}
			defer tx.Rollback(ctx)
			return tx.QueryRow(ctx, `SELECT village_id::text, privacy_epoch FROM ttg.sessions WHERE id = $1`, payload.SessionID).Scan(&villageID, &epoch)
		}()
		if err != nil {
			// A session absent from the restored backup has nothing to erase.
			skippedCount++
			continue
		}
		if epoch > payload.PrivacyEpoch {
			skippedCount++
			continue
		}
		if err := func() error {
			tx, err := c.Pool.Begin(ctx)
			if err != nil {
				return err
			}
			defer tx.Rollback(ctx)
			if _, err := tx.Exec(ctx, `INSERT INTO ttg.privacy_tombstones (id, actor_ref_protected, scope) VALUES ($1::uuid, $2, $3) ON CONFLICT (id) DO NOTHING`, payload.TombstoneID, payload.ActorRefProtected, payload.Scope); err != nil {
				return err
			}
			return tx.Commit(ctx)
		}(); err != nil {
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
		result, err := red.Anonymize(ctx, lease, payload.SessionID, []string{payload.ActorRefProtected})
		if err != nil {
			return RestoreResult{}, err
		}
		applied = append(applied, payload.TombstoneID)
		appliedCount++
		if err := c.recordRestore(ctx, proof, targetIdentity, deletion, payload.SessionID, result.Revision); err != nil {
			return RestoreResult{}, err
		}
	}
	if appliedCount == 0 {
		return RestoreResult{LedgerID: proof.LedgerID, LastSeq: proof.LastSeq, HeadHash: proof.HeadHash, Skipped: skippedCount, AppliedErasures: applied}, nil
	}
	runID, err := c.finishRun(ctx, proof, targetIdentity, applied)
	if err != nil {
		return RestoreResult{}, err
	}
	return RestoreResult{LedgerID: proof.LedgerID, LastSeq: proof.LastSeq, HeadHash: proof.HeadHash, Applied: appliedCount, Skipped: skippedCount, RunID: runID, AppliedErasures: applied}, nil
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

func (c *RestoreCoordinator) recordRestore(ctx context.Context, proof CoverageProof, target string, deletion Deletion, sessionID string, revision int64) error {
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var runID string
	err = tx.QueryRow(ctx, `INSERT INTO ttg.privacy_restore_runs (id, ledger_id, ledger_last_seq, ledger_head_hash, target_identity, result_hash)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
		ON CONFLICT (ledger_id, ledger_last_seq, target_identity) DO UPDATE SET target_identity = EXCLUDED.target_identity
		RETURNING id::text`, proof.LedgerID, proof.LastSeq, proof.HeadHash, target, hashString(proof.HeadHash+deletion.Payload.TombstoneID)).Scan(&runID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.privacy_restore_erasures (run_id, tombstone_id, session_id, actor_ref_protected, applied_revision)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, runID, deletion.Payload.TombstoneID, sessionID, deletion.Payload.ActorRefProtected, revision); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (c *RestoreCoordinator) finishRun(ctx context.Context, proof CoverageProof, target string, tombstones []string) (string, error) {
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var runID string
	err = tx.QueryRow(ctx, `INSERT INTO ttg.privacy_restore_runs (id, ledger_id, ledger_last_seq, ledger_head_hash, target_identity, result_hash)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
		ON CONFLICT (ledger_id, ledger_last_seq, target_identity) DO UPDATE SET target_identity = EXCLUDED.target_identity
		RETURNING id::text`, proof.LedgerID, proof.LastSeq, proof.HeadHash, target, hashString(proof.HeadHash)).Scan(&runID)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return runID, nil
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
