package privacy

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/reducer"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
)

var (
	ErrInvalidSubject  = errors.New("privacy: invalid protected subject reference")
	ErrNotAcknowledged = errors.New("privacy: deletion ledger has not acknowledged the request")
	ErrErasureState    = errors.New("privacy: erasure request state is invalid")
)

type Service struct {
	Pool    *storage.Pool
	Ledger  *Ledger
	Reducer *reducer.Reducer
}

type ErasureRequest struct {
	ID           string
	SessionID    string
	ActorRef     string
	Scope        string
	Status       string
	PrivacyEpoch int64
	OutboxSeq    int64
	LedgerAckRef string
}

func (s *Service) Request(ctx context.Context, sessionID, actorRef, reason, requestedBy string) (ErasureRequest, error) {
	if s == nil || s.Pool == nil || !validActorRef(actorRef) {
		return ErasureRequest{}, ErrInvalidSubject
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return ErasureRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var villageID string
	var epoch int64
	if err := tx.QueryRow(ctx, `SELECT village_id::text, privacy_epoch FROM ttg.sessions WHERE id = $1 FOR UPDATE`, sessionID).Scan(&villageID, &epoch); err != nil {
		return ErasureRequest{}, err
	}
	var existing ErasureRequest
	err = tx.QueryRow(ctx, `SELECT id::text, session_id::text, actor_ref_protected, scope, status, privacy_epoch
		FROM ttg.privacy_erase_requests WHERE session_id = $1 AND actor_ref_protected = $2 AND scope = 'session'`, sessionID, actorRef).Scan(
		&existing.ID, &existing.SessionID, &existing.ActorRef, &existing.Scope, &existing.Status, &existing.PrivacyEpoch,
	)
	if err == nil {
		return existing, tx.Commit(ctx)
	}
	var tombstoneID, outboxID string
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.privacy_tombstones (id, actor_ref_protected, scope) VALUES (gen_random_uuid(), $1, 'session') RETURNING id::text`, actorRef).Scan(&tombstoneID); err != nil {
		return ErasureRequest{}, err
	}
	payload, err := json.Marshal(LedgerPayload{TombstoneID: tombstoneID, SessionID: sessionID, Scope: "session", ActorRefProtected: actorRef, PrivacyEpoch: epoch, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return ErasureRequest{}, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.privacy_ledger_outbox (id, tombstone_id, payload) VALUES (gen_random_uuid(), $1, $2::jsonb) RETURNING id::text, seq`, tombstoneID, payload).Scan(&outboxID, &existing.OutboxSeq); err != nil {
		return ErasureRequest{}, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.privacy_erase_requests (id, session_id, actor_ref_protected, reason, status, privacy_epoch, requested_by, scope)
		VALUES (gen_random_uuid(), $1, $2, $3, 'queued', $4, NULLIF($5, '')::uuid, 'session') RETURNING id::text`, sessionID, actorRef, reason, epoch, requestedBy).Scan(&existing.ID); err != nil {
		return ErasureRequest{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE ttg.players SET hide_alias = true, alias = NULL WHERE village_id = $1 AND actor_key = $2`, villageID, actorRef); err != nil {
		return ErasureRequest{}, err
	}
	existing.SessionID = sessionID
	existing.ActorRef = actorRef
	existing.Scope = "session"
	existing.Status = "queued"
	existing.PrivacyEpoch = epoch
	if err := tx.Commit(ctx); err != nil {
		return ErasureRequest{}, err
	}
	return existing, nil
}

func (s *Service) Export(ctx context.Context, limit int) (int, error) {
	if s == nil || s.Ledger == nil {
		return 0, ErrLedgerKey
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	readTx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	rows, err := readTx.Query(ctx, `SELECT id::text, seq, payload FROM ttg.privacy_ledger_outbox WHERE exported_at IS NULL ORDER BY seq LIMIT $1`, limit)
	if err != nil {
		_ = readTx.Rollback(ctx)
		return 0, err
	}
	type pending struct {
		ID      string
		Seq     int64
		Payload []byte
	}
	items := make([]pending, 0)
	for rows.Next() {
		var item pending
		if err := rows.Scan(&item.ID, &item.Seq, &item.Payload); err != nil {
			rows.Close()
			_ = readTx.Rollback(ctx)
			return 0, err
		}
		items = append(items, item)
	}
	rows.Close()
	_ = readTx.Rollback(ctx)
	exported := 0
	for _, item := range items {
		var payload LedgerPayload
		if err := json.Unmarshal(item.Payload, &payload); err != nil {
			return exported, err
		}
		record, err := s.Ledger.Append(payload)
		if err != nil {
			if failTx, failErr := s.Pool.Begin(ctx); failErr == nil {
				_, _ = failTx.Exec(ctx, `UPDATE ttg.privacy_ledger_outbox SET attempts = attempts + 1, last_error = $2 WHERE id = $1`, item.ID, err.Error())
				_ = failTx.Commit(ctx)
			}
			return exported, err
		}
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return exported, err
		}
		if _, err := tx.Exec(ctx, `UPDATE ttg.privacy_ledger_outbox SET exported_at = now(), ledger_ack_ref = $2, attempts = attempts + 1, last_error = NULL WHERE id = $1 AND exported_at IS NULL`, item.ID, record.AckRef); err != nil {
			_ = tx.Rollback(ctx)
			return exported, err
		}
		if err := tx.Commit(ctx); err != nil {
			return exported, err
		}
		exported++
	}
	return exported, nil
}

// Apply must be called with a valid W0401 lease. It refuses to erase until the
// external ledger has acknowledged the tombstone, then commits the anonymized
// checkpoint and SQL erasure through the fenced reducer.
func (s *Service) Apply(ctx context.Context, lease writer.Lease, requestID string) (reducer.Result, error) {
	var sessionID, actorRef, status string
	var ack *string
	if err := func() error {
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		return tx.QueryRow(ctx, `SELECT r.session_id::text, r.actor_ref_protected, r.status, o.ledger_ack_ref
			FROM ttg.privacy_erase_requests r
			JOIN ttg.privacy_tombstones t ON t.actor_ref_protected = r.actor_ref_protected AND t.scope = r.scope
			JOIN ttg.privacy_ledger_outbox o ON o.tombstone_id = t.id
			WHERE r.id = $1 ORDER BY o.seq DESC LIMIT 1`, requestID).Scan(&sessionID, &actorRef, &status, &ack)
	}(); err != nil {
		return reducer.Result{}, err
	}
	if ack == nil || *ack == "" {
		return reducer.Result{}, ErrNotAcknowledged
	}
	if status == "applied" {
		return reducer.Result{}, nil
	}
	if status != "queued" {
		return reducer.Result{}, ErrErasureState
	}
	actors, err := s.actorKeys(ctx, sessionID, actorRef)
	if err != nil {
		return reducer.Result{}, err
	}
	result, err := s.Reducer.Anonymize(ctx, lease, sessionID, actors)
	if err != nil {
		return reducer.Result{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return reducer.Result{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE ttg.privacy_erase_requests SET status = 'applied', applied_at = now() WHERE id = $1 AND status = 'queued'`, requestID); err != nil {
		return reducer.Result{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE ttg.privacy_tombstones SET restore_applied_at = now() WHERE actor_ref_protected = $1 AND scope = 'session'`, actorRef); err != nil {
		return reducer.Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return reducer.Result{}, err
	}
	return result, nil
}

func (s *Service) actorKeys(ctx context.Context, sessionID, actorRef string) ([]string, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `SELECT DISTINCT a.actor_key::text FROM ttg.actor_identity_aliases a
		WHERE a.village_id = (SELECT village_id FROM ttg.sessions WHERE id = $1)
		AND a.player_id IN (SELECT player_id FROM ttg.actor_identity_aliases WHERE village_id = a.village_id AND actor_key = $2)`, sessionID, actorRef)
	if err != nil {
		return nil, err
	}
	keys := []string{actorRef}
	seen := map[string]struct{}{actorRef: {}}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return nil, err
		}
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	rows.Close()
	return keys, nil
}

func validActorRef(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func (s *Service) Status(ctx context.Context, requestID string) (ErasureRequest, error) {
	var request ErasureRequest
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return request, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `SELECT r.id::text, r.session_id::text, r.actor_ref_protected, r.scope, r.status, r.privacy_epoch,
		COALESCE(o.seq, 0), COALESCE(o.ledger_ack_ref, '')
		FROM ttg.privacy_erase_requests r
		LEFT JOIN ttg.privacy_tombstones t ON t.actor_ref_protected = r.actor_ref_protected AND t.scope = r.scope
		LEFT JOIN ttg.privacy_ledger_outbox o ON o.tombstone_id = t.id
		WHERE r.id = $1 ORDER BY o.seq DESC LIMIT 1`, requestID).Scan(
		&request.ID, &request.SessionID, &request.ActorRef, &request.Scope, &request.Status, &request.PrivacyEpoch, &request.OutboxSeq, &request.LedgerAckRef)
	if err != nil {
		return ErasureRequest{}, err
	}
	return request, nil
}

func (s *Service) Coverage(ctx context.Context, target string, horizon time.Time, required []string) error {
	if s == nil || s.Ledger == nil {
		return ErrLedgerCoverage
	}
	proof, err := s.Ledger.Verify(target, horizon)
	if err != nil {
		return err
	}
	return AllowClientServing(proof, horizon, required)
}
