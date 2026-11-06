package privacy

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/reducer"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
	"github.com/jackc/pgx/v5"
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
	TombstoneID  string
	ParentID     string
}

type ScopeRequest struct {
	ID        string
	VillageID string
	ActorRef  string
	Scope     string
	Status    string
	Children  []ErasureRequest
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
	var existingTombstone *string
	err = tx.QueryRow(ctx, `SELECT id::text, session_id::text, actor_ref_protected, scope, status, privacy_epoch, tombstone_id::text
		FROM ttg.privacy_erase_requests WHERE session_id = $1 AND actor_ref_protected = $2 AND scope = 'session'`, sessionID, actorRef).Scan(
		&existing.ID, &existing.SessionID, &existing.ActorRef, &existing.Scope, &existing.Status, &existing.PrivacyEpoch, &existingTombstone,
	)
	if existingTombstone != nil {
		existing.TombstoneID = *existingTombstone
	}
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
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.privacy_erase_requests (id, session_id, actor_ref_protected, reason, status, privacy_epoch, requested_by, scope, tombstone_id)
		VALUES (gen_random_uuid(), $1, $2, $3, 'queued', $4, NULLIF($5, '')::uuid, 'session', $6) RETURNING id::text`, sessionID, actorRef, reason, epoch, requestedBy, tombstoneID).Scan(&existing.ID); err != nil {
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
	existing.TombstoneID = tombstoneID
	if err := tx.Commit(ctx); err != nil {
		return ErasureRequest{}, err
	}
	return existing, nil
}

// RequestScope keeps the old session/village entry point. Village scope now creates a
// durable parent and a complete child inventory atomically.
func (s *Service) RequestScope(ctx context.Context, sessionID, actorRef, reason, scope string) ([]ErasureRequest, error) {
	if scope == "" || scope == "session" {
		request, err := s.Request(ctx, sessionID, actorRef, reason, "")
		return []ErasureRequest{request}, err
	}
	if scope != "village" {
		return nil, ErrErasureState
	}
	parent, err := s.CreateVillageRequest(ctx, sessionID, actorRef, reason, "")
	if err != nil {
		return nil, err
	}
	return parent.Children, nil
}

func (s *Service) CreateVillageRequest(ctx context.Context, sessionID, actorRef, reason, requestedBy string) (ScopeRequest, error) {
	if s == nil || s.Pool == nil || !validActorRef(actorRef) {
		return ScopeRequest{}, ErrInvalidSubject
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return ScopeRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var villageID string
	if err := tx.QueryRow(ctx, `SELECT village_id::text FROM ttg.sessions WHERE id = $1 FOR UPDATE`, sessionID).Scan(&villageID); err != nil {
		return ScopeRequest{}, err
	}
	var existing ScopeRequest
	var existingID string
	err = tx.QueryRow(ctx, `SELECT id::text, status FROM ttg.privacy_scope_requests WHERE village_id = $1 AND actor_ref_protected = $2 AND scope = 'village'`, villageID, actorRef).Scan(&existingID, &existing.Status)
	if err == nil {
		existing.ID, existing.VillageID, existing.ActorRef, existing.Scope = existingID, villageID, actorRef, "village"
		existing.Children, err = s.children(ctx, tx, existingID)
		if err != nil {
			return ScopeRequest{}, err
		}
		return existing, tx.Commit(ctx)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.privacy_scope_requests (id, village_id, actor_ref_protected, scope, status) VALUES (gen_random_uuid(), $1, $2, 'village', 'queued') RETURNING id::text`, villageID, actorRef).Scan(&existing.ID); err != nil {
		return ScopeRequest{}, err
	}
	// Suppression is immediate for the whole village.
	if _, err := tx.Exec(ctx, `UPDATE ttg.players SET hide_alias = true, alias = NULL WHERE village_id = $1 AND actor_key = $2`, villageID, actorRef); err != nil {
		return ScopeRequest{}, err
	}
	rows, err := tx.Query(ctx, `SELECT DISTINCT s.id::text FROM ttg.sessions s
		WHERE s.village_id = $1 AND (
			EXISTS (SELECT 1 FROM ttg.wave_members w WHERE w.session_id = s.id AND w.actor_key = $2)
			OR EXISTS (SELECT 1 FROM ttg.session_player_progress p WHERE p.session_id = s.id AND p.actor_key = $2)
			OR EXISTS (SELECT 1 FROM ttg.input_events i WHERE i.session_id = s.id AND i.actor_key = $2)
		) ORDER BY s.id::text`, villageID, actorRef)
	if err != nil {
		return ScopeRequest{}, err
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return ScopeRequest{}, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	// If the subject exists only as a village player/alias, still create the active session child.
	if len(ids) == 0 {
		var active string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM ttg.sessions WHERE village_id = $1 AND lifecycle <> 'stopped' LIMIT 1`, villageID).Scan(&active); err == nil {
			ids = append(ids, active)
		}
	}
	for _, childSession := range ids {
		child, err := s.createChildTx(ctx, tx, childSession, villageID, actorRef, reason, requestedBy, existing.ID)
		if err != nil {
			return ScopeRequest{}, err
		}
		existing.Children = append(existing.Children, child)
	}
	existing.VillageID, existing.ActorRef, existing.Scope, existing.Status = villageID, actorRef, "village", "queued"
	if err := tx.Commit(ctx); err != nil {
		return ScopeRequest{}, err
	}
	return existing, nil
}

func (s *Service) createChildTx(ctx context.Context, tx pgx.Tx, sessionID, villageID, actorRef, reason, requestedBy, parentID string) (ErasureRequest, error) {
	var epoch int64
	if err := tx.QueryRow(ctx, `SELECT privacy_epoch FROM ttg.sessions WHERE id = $1 FOR UPDATE`, sessionID).Scan(&epoch); err != nil {
		return ErasureRequest{}, err
	}
	var tombstoneID string
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.privacy_tombstones (id, actor_ref_protected, scope) VALUES (gen_random_uuid(), $1, 'village') RETURNING id::text`, actorRef).Scan(&tombstoneID); err != nil {
		return ErasureRequest{}, err
	}
	payload, err := json.Marshal(LedgerPayload{TombstoneID: tombstoneID, SessionID: sessionID, Scope: "village", ActorRefProtected: actorRef, PrivacyEpoch: epoch, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return ErasureRequest{}, err
	}
	var seq int64
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.privacy_ledger_outbox (id, tombstone_id, payload) VALUES (gen_random_uuid(), $1, $2::jsonb) RETURNING seq`, tombstoneID, payload).Scan(&seq); err != nil {
		return ErasureRequest{}, err
	}
	var request ErasureRequest
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.privacy_erase_requests (id, session_id, actor_ref_protected, reason, status, privacy_epoch, requested_by, scope, tombstone_id, parent_id)
		VALUES (gen_random_uuid(), $1, $2, $3, 'queued', $4, NULLIF($5, '')::uuid, 'village', $6, $7) RETURNING id::text`, sessionID, actorRef, reason, epoch, requestedBy, tombstoneID, parentID).Scan(&request.ID); err != nil {
		return ErasureRequest{}, err
	}
	request.SessionID, request.ActorRef, request.Scope, request.Status, request.PrivacyEpoch = sessionID, actorRef, "village", "queued", epoch
	request.OutboxSeq, request.TombstoneID, request.ParentID = seq, tombstoneID, parentID
	return request, nil
}

func (s *Service) children(ctx context.Context, tx pgx.Tx, parentID string) ([]ErasureRequest, error) {
	rows, err := tx.Query(ctx, `SELECT r.id::text, r.session_id::text, r.actor_ref_protected, r.scope, r.status, r.privacy_epoch,
		COALESCE(o.seq,0), COALESCE(o.ledger_ack_ref,''), COALESCE(r.tombstone_id::text,''), COALESCE(r.parent_id::text,'')
		FROM ttg.privacy_erase_requests r LEFT JOIN ttg.privacy_ledger_outbox o ON o.tombstone_id = r.tombstone_id
		WHERE r.parent_id = $1 ORDER BY r.session_id`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ErasureRequest, 0)
	for rows.Next() {
		var r ErasureRequest
		if err := rows.Scan(&r.ID, &r.SessionID, &r.ActorRef, &r.Scope, &r.Status, &r.PrivacyEpoch, &r.OutboxSeq, &r.LedgerAckRef, &r.TombstoneID, &r.ParentID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ApplyScope resumes all pending children and only marks the parent applied after every
// required child is applied.
func (s *Service) ApplyScope(ctx context.Context, lease writer.Lease, parentID string) (ScopeRequest, error) {
	var parent ScopeRequest
	var villageID string
	if err := func() error {
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		if err := tx.QueryRow(ctx, `SELECT id::text, village_id::text, actor_ref_protected, scope, status FROM ttg.privacy_scope_requests WHERE id = $1 FOR UPDATE`, parentID).Scan(&parent.ID, &villageID, &parent.ActorRef, &parent.Scope, &parent.Status); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}(); err != nil {
		return ScopeRequest{}, err
	}
	parent.VillageID = villageID
	if parent.Status == "applied" {
		return parent, nil
	}
	children, err := s.scopeChildren(ctx, parentID)
	if err != nil {
		return ScopeRequest{}, err
	}
	for _, child := range children {
		if child.Status == "applied" {
			continue
		}
		if _, err := s.Apply(ctx, lease, child.ID); err != nil {
			if markTx, markErr := s.Pool.Begin(ctx); markErr == nil {
				_, _ = markTx.Exec(ctx, `UPDATE ttg.privacy_scope_requests SET status = 'blocked' WHERE id = $1`, parentID)
				_ = markTx.Commit(ctx)
			}
			return ScopeRequest{}, err
		}
	}
	if err := func() error {
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		var pending int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM ttg.privacy_erase_requests WHERE parent_id = $1 AND status <> 'applied'`, parentID).Scan(&pending); err != nil {
			return err
		}
		if pending != 0 {
			return ErrErasureState
		}
		if _, err := tx.Exec(ctx, `UPDATE ttg.privacy_scope_requests SET status = 'applied', completed_at = now() WHERE id = $1`, parentID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}(); err != nil {
		return ScopeRequest{}, err
	}
	parent.Status = "applied"
	parent.Children, _ = s.scopeChildren(ctx, parentID)
	return parent, nil
}

func (s *Service) scopeChildren(ctx context.Context, parentID string) ([]ErasureRequest, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	return s.children(ctx, tx, parentID)
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
	var sessionID, actorRef, status, tombstoneID string
	var ack *string
	if err := func() error {
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		return tx.QueryRow(ctx, `SELECT r.session_id::text, r.actor_ref_protected, r.status, r.tombstone_id::text, o.ledger_ack_ref
			FROM ttg.privacy_erase_requests r
			JOIN ttg.privacy_ledger_outbox o ON o.tombstone_id = r.tombstone_id
			WHERE r.id = $1 AND r.tombstone_id IS NOT NULL ORDER BY o.seq DESC LIMIT 1`, requestID).Scan(&sessionID, &actorRef, &status, &tombstoneID, &ack)
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
	result, err := s.Reducer.AnonymizeWithCompletion(ctx, lease, sessionID, actors, func(ctx context.Context, tx writer.FencedTx, _ int64) error {
		tag, err := tx.Exec(ctx, `UPDATE ttg.privacy_erase_requests SET status = 'applied', applied_at = now() WHERE id = $1 AND status = 'queued' AND tombstone_id = $2`, requestID, tombstoneID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrErasureState
		}
		_, err = tx.Exec(ctx, `UPDATE ttg.privacy_tombstones SET restore_applied_at = now() WHERE id = $1`, tombstoneID)
		return err
	})
	if err != nil {
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
		COALESCE(o.seq, 0), COALESCE(o.ledger_ack_ref, ''), COALESCE(r.tombstone_id::text, '')
		FROM ttg.privacy_erase_requests r
		LEFT JOIN ttg.privacy_ledger_outbox o ON o.tombstone_id = r.tombstone_id
		WHERE r.id = $1 ORDER BY o.seq DESC LIMIT 1`, requestID).Scan(
		&request.ID, &request.SessionID, &request.ActorRef, &request.Scope, &request.Status, &request.PrivacyEpoch, &request.OutboxSeq, &request.LedgerAckRef, &request.TombstoneID)
	if err != nil {
		return ErasureRequest{}, err
	}
	return request, nil
}

func (s *Service) Coverage(ctx context.Context, target string, horizon time.Time, required []string) error {
	if s == nil || s.Ledger == nil {
		return ErrLedgerCoverage
	}
	proof, err := s.Ledger.Verify(target)
	if err != nil {
		return err
	}
	return AllowClientServing(proof, horizon, required)
}
