package privacy

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/identity"
	"github.com/hung98dev/ttgqldl/backend/internal/reducer"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
)

func testPool(t *testing.T) *storage.Pool {
	t.Helper()
	dsn := os.Getenv("TTG_TEST_DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		if os.Getenv("TTG_REQUIRE_DATABASE") == "1" {
			t.Fatal("TTG_TEST_DATABASE_URL required")
		}
		t.Skip("TTG_TEST_DATABASE_URL not set")
	}
	pool, err := storage.Open(context.Background(), storage.Options{DSN: dsn, MaxConns: 8, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func setupPrivacySession(t *testing.T, pool *storage.Pool) (villageID, sessionID, actor string, manager *writer.Manager, red *reducer.Reducer) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.villages (id, broadcaster_key) VALUES (gen_random_uuid(), md5(random()::text) || md5(random()::text)) RETURNING id::text`).Scan(&villageID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version) VALUES (gen_random_uuid(), $1, 'mock', 'waiting', 'ttg-rules-v1') RETURNING id::text`, villageID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	actor = strings.Repeat("a", 64)
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.input_events (session_id, input_seq, accepted_tick, scheduled_tick, wave_no, accepted_wave, actor_key, kind, canonical_payload, rules_version) VALUES ($1, 1, -1, 0, 1, 1, $2, 'role', '{"kind":"role","role":"builder"}'::jsonb, 'ttg-rules-v1')`, sessionID, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE ttg.sessions SET input_sequence = 1 WHERE id = $1`, sessionID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	manager, err = writer.New(pool, "privacy-test", writer.Config{LeaseDuration: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	red, err = reducer.New(manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := manager.Acquire(ctx, villageID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := red.Step(ctx, lease, sessionID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		tx, err := pool.Begin(cleanup)
		if err != nil {
			return
		}
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_ledger_outbox WHERE tombstone_id IN (SELECT tombstone_id FROM ttg.privacy_erase_requests WHERE session_id = $1)`, sessionID)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_erase_requests WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_tombstones t WHERE t.actor_ref_protected = $1
			AND NOT EXISTS (SELECT 1 FROM ttg.privacy_ledger_outbox o WHERE o.tombstone_id = t.id)
			AND NOT EXISTS (SELECT 1 FROM ttg.privacy_erase_requests r WHERE r.tombstone_id = t.id)`, actor)
		for _, statement := range []string{
			`DELETE FROM ttg.stream_outbox WHERE session_id = $1`,
			`DELETE FROM ttg.domain_events WHERE session_id = $1`,
			`DELETE FROM ttg.checkpoints WHERE session_id = $1`,
			`DELETE FROM ttg.reducer_transitions WHERE session_id = $1`,
			`DELETE FROM ttg.input_receipts WHERE session_id = $1`,
			`DELETE FROM ttg.input_events WHERE session_id = $1`,
			`DELETE FROM ttg.wave_members WHERE session_id = $1`,
			`DELETE FROM ttg.ballots WHERE session_id = $1`,
			`DELETE FROM ttg.session_player_progress WHERE session_id = $1`,
			`DELETE FROM ttg.sessions WHERE id = $1`,
		} {
			_, _ = tx.Exec(cleanup, statement, sessionID)
		}
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.actor_identity_aliases WHERE village_id = $1`, villageID)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.players WHERE village_id = $1`, villageID)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.writer_leases WHERE village_id = $1`, villageID)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.villages WHERE id = $1`, villageID)
		_ = tx.Commit(cleanup)
	})
	return villageID, sessionID, actor, manager, red
}

func TestErasureRequiresLedgerThenAnonymizes(t *testing.T) {
	pool := testPool(t)
	villageID, sessionID, actor, manager, red := setupPrivacySession(t, pool)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var playerID string
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.players (id, village_id, actor_key, ephemeral, alias) VALUES (gen_random_uuid(), $1, $2, false, 'Alice') RETURNING id::text`, villageID, actor).Scan(&playerID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id) VALUES ($1, $2, 'v1', $3)`, villageID, actor, playerID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.session_player_progress (session_id, actor_key, session_xp) VALUES ($1, $2, 42)`, sessionID, actor); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	enc := make([]byte, 32)
	chain := make([]byte, 32)
	_, _ = rand.Read(enc)
	_, _ = rand.Read(chain)
	ledger, err := OpenLedger(t.TempDir(), "ledger-test", enc, chain)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{Pool: pool, Ledger: ledger, Reducer: red}
	request, err := service.Request(ctx, sessionID, actor, "synthetic erasure", "")
	if err != nil {
		t.Fatal(err)
	}
	if request.Status != "queued" || request.OutboxSeq == 0 {
		t.Fatalf("request %+v", request)
	}
	if got := boolValue(t, pool, `SELECT hide_alias FROM ttg.players WHERE id = $1`, playerID); !got {
		t.Fatal("alias not suppressed")
	}
	lease, err := manager.Acquire(ctx, villageID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Apply(ctx, lease, request.ID); err == nil {
		t.Fatal("erasure applied before ledger acknowledgment")
	}
	if _, err := service.Export(ctx, 10); err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(ctx, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.LedgerAckRef == "" {
		t.Fatalf("no ack: %+v", status)
	}
	if _, err := service.Apply(ctx, lease, request.ID); err != nil {
		t.Fatal(err)
	}
	if got := sessionValue(t, pool, `SELECT status FROM ttg.privacy_erase_requests WHERE id = $1`, request.ID); got != "applied" {
		t.Fatalf("status=%s", got)
	}
	if got := sessionValue(t, pool, `SELECT count(*)::text FROM ttg.actor_identity_aliases WHERE village_id = $1`, villageID); got != "0" {
		t.Fatalf("aliases=%s", got)
	}
	if got := sessionValue(t, pool, `SELECT count(*)::text FROM ttg.players WHERE village_id = $1`, villageID); got != "0" {
		t.Fatalf("players=%s", got)
	}
	if got := sessionValue(t, pool, `SELECT privacy_epoch::text FROM ttg.sessions WHERE id = $1`, sessionID); got != "1" {
		t.Fatalf("epoch=%s", got)
	}
	proof, err := ledger.Verify("db-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := AllowClientServing(proof, time.Now().Add(-time.Minute), []string{request.ID}); err == nil {
		t.Fatal("required tombstone id should not match ledger tombstone")
	}
	if _, err := red.VerifyReplay(ctx, sessionID, 0); err != nil {
		t.Fatalf("replay after erasure: %v", err)
	}
	_ = domain.RulesVersion
}

func sessionValue(t *testing.T, pool *storage.Pool, query string, args ...any) string {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var value string
	if err := tx.QueryRow(ctx, query, args...).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}
func boolValue(t *testing.T, pool *storage.Pool, query string, args ...any) bool {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var value bool
	if err := tx.QueryRow(ctx, query, args...).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestAliasRotationResolvesSamePlayerAndErasureRemovesAllVersions(t *testing.T) {
	pool := testPool(t)
	villageID, sessionID, _, manager, red := setupPrivacySession(t, pool)
	ctx := context.Background()
	v1, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	v2, err := identity.NewKey("v2", []byte("fedcba9876543210fedcba9876543210"))
	if err != nil {
		t.Fatal(err)
	}
	v3, err := identity.NewKey("v3", []byte("00112233445566778899aabbccddeeff"))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewKeySet(v3, []identity.Key{v2, v1})
	if err != nil {
		t.Fatal(err)
	}
	input := identity.Input{Provider: "tiktok", ProviderVersion: "v1", BroadcasterKey: "synthetic-broadcaster", ViewerID: "viewer-rotation-1"}
	candidates, err := keys.Candidates(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 3 {
		t.Fatalf("candidates=%d", len(candidates))
	}
	oldKey := string(candidates[2].ActorKey)
	newKey := string(candidates[1].ActorKey)
	activeKey := string(candidates[0].ActorKey)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tx, err := pool.Begin(cleanup)
		if err != nil {
			return
		}
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_ledger_outbox WHERE tombstone_id IN (SELECT id FROM ttg.privacy_tombstones WHERE actor_ref_protected = $1)`, oldKey)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_erase_requests WHERE actor_ref_protected = $1`, oldKey)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_tombstones WHERE actor_ref_protected = $1`, oldKey)
		_ = tx.Commit(cleanup)
	})
	var playerID string
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.players (id, village_id, actor_key, ephemeral, alias) VALUES (gen_random_uuid(), $1, $2, false, 'RotationSubject') RETURNING id::text`, villageID, oldKey).Scan(&playerID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id) VALUES ($1, $2, 'v1', $3), ($1, $4, 'v2', $3), ($1, $5, 'v3', $3)`, villageID, oldKey, playerID, newKey, activeKey); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var resolved string
	if err := func() error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		return tx.QueryRow(ctx, `SELECT player_id::text FROM ttg.actor_identity_aliases WHERE village_id = $1 AND actor_key = $2`, villageID, oldKey).Scan(&resolved)
	}(); err != nil {
		t.Fatal(err)
	}
	if resolved != playerID {
		t.Fatalf("rotation resolved %s want %s", resolved, playerID)
	}
	enc := make([]byte, 32)
	chain := make([]byte, 32)
	_, _ = rand.Read(enc)
	_, _ = rand.Read(chain)
	ledger, err := OpenLedger(t.TempDir(), "rotation-ledger", enc, chain)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{Pool: pool, Ledger: ledger, Reducer: red}
	request, err := service.Request(ctx, sessionID, oldKey, "rotation erasure", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Export(ctx, 10); err != nil {
		t.Fatal(err)
	}
	lease, err := manager.Acquire(ctx, villageID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Apply(ctx, lease, request.ID); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := func() error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		return tx.QueryRow(ctx, `SELECT count(*) FROM ttg.actor_identity_aliases WHERE village_id = $1`, villageID).Scan(&remaining)
	}(); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("aliases survived erasure: %d", remaining)
	}
	var after int
	if err := func() error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		return tx.QueryRow(ctx, `SELECT count(*) FROM ttg.actor_identity_aliases WHERE village_id = $1 AND actor_key = ANY($2)`, villageID, []string{oldKey, newKey, activeKey}).Scan(&after)
	}(); err != nil {
		t.Fatal(err)
	}
	if after != 0 {
		t.Fatalf("old/new aliases survived: %d", after)
	}
}

func TestErasureCannotBorrowAnotherRequestsAck(t *testing.T) {
	pool := testPool(t)
	_, sessionOne, actor, _, redOne := setupPrivacySession(t, pool)
	_, sessionTwo, _, managerTwo, redTwo := setupPrivacySession(t, pool)
	enc := make([]byte, 32)
	chain := make([]byte, 32)
	_, _ = rand.Read(enc)
	_, _ = rand.Read(chain)
	ledger, err := OpenLedger(t.TempDir(), "binding-ledger", enc, chain)
	if err != nil {
		t.Fatal(err)
	}
	serviceOne := &Service{Pool: pool, Ledger: ledger, Reducer: redOne}
	requestOne, err := serviceOne.Request(context.Background(), sessionOne, actor, "request one", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := serviceOne.Export(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	serviceTwo := &Service{Pool: pool, Ledger: ledger, Reducer: redTwo}
	requestTwo, err := serviceTwo.Request(context.Background(), sessionTwo, actor, "request two", "")
	if err != nil {
		t.Fatal(err)
	}
	if requestOne.TombstoneID == "" || requestTwo.TombstoneID == "" || requestOne.TombstoneID == requestTwo.TombstoneID {
		t.Fatalf("tombstone binding missing: %+v %+v", requestOne, requestTwo)
	}
	leaseTwo, err := managerTwo.Acquire(context.Background(), mustVillage(t, pool, sessionTwo))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := serviceTwo.Apply(context.Background(), leaseTwo, requestTwo.ID); !errors.Is(err, ErrNotAcknowledged) {
		t.Fatalf("borrowed ack result = %v", err)
	}
}

func mustVillage(t *testing.T, pool *storage.Pool, sessionID string) string {
	t.Helper()
	return sessionValue(t, pool, `SELECT village_id::text FROM ttg.sessions WHERE id = $1`, sessionID)
}

func TestVillageScopeParentIdempotentAndResumable(t *testing.T) {
	pool := testPool(t)
	villageID, sessionID, actor, manager, red := setupPrivacySession(t, pool)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var playerID string
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.players (id, village_id, actor_key, ephemeral, alias) VALUES (gen_random_uuid(), $1, $2, false, 'VillageParent') RETURNING id::text`, villageID, actor).Scan(&playerID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.actor_identity_aliases (village_id, actor_key, key_version, player_id) VALUES ($1, $2, 'v1', $3)`, villageID, actor, playerID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	enc := make([]byte, 32)
	chain := make([]byte, 32)
	_, _ = rand.Read(enc)
	_, _ = rand.Read(chain)
	ledger, err := OpenLedger(t.TempDir(), "scope-ledger", enc, chain)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{Pool: pool, Ledger: ledger, Reducer: red}
	parent, err := service.CreateVillageRequest(ctx, sessionID, actor, "village scope", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tx, err := pool.Begin(cleanup)
		if err != nil {
			return
		}
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_ledger_outbox WHERE tombstone_id IN (SELECT tombstone_id FROM ttg.privacy_erase_requests WHERE parent_id = $1)`, parent.ID)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_erase_requests WHERE parent_id = $1`, parent.ID)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_tombstones t WHERE t.actor_ref_protected = $1 AND NOT EXISTS (SELECT 1 FROM ttg.privacy_ledger_outbox o WHERE o.tombstone_id = t.id) AND NOT EXISTS (SELECT 1 FROM ttg.privacy_erase_requests r WHERE r.tombstone_id = t.id)`, actor)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_scope_requests WHERE id = $1`, parent.ID)
		_ = tx.Commit(cleanup)
	})
	if parent.Status != "queued" || len(parent.Children) != 1 || parent.Children[0].ParentID != parent.ID {
		t.Fatalf("parent %+v", parent)
	}
	again, err := service.CreateVillageRequest(ctx, sessionID, actor, "village scope", "")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != parent.ID || len(again.Children) != 1 {
		t.Fatalf("idempotent parent %+v", again)
	}
	lease, err := manager.Acquire(ctx, villageID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplyScope(ctx, lease, parent.ID); !errors.Is(err, ErrNotAcknowledged) {
		t.Fatalf("unexported parent apply = %v", err)
	}
	if _, err := service.Export(ctx, 10); err != nil {
		t.Fatal(err)
	}
	resumed, err := service.ApplyScope(ctx, lease, parent.ID)
	if err != nil {
		t.Fatalf("resume parent: %v", err)
	}
	if resumed.Status != "applied" {
		t.Fatalf("parent status=%s", resumed.Status)
	}
	var aliases int
	if err := func() error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		return tx.QueryRow(ctx, `SELECT count(*) FROM ttg.actor_identity_aliases WHERE village_id = $1`, villageID).Scan(&aliases)
	}(); err != nil {
		t.Fatal(err)
	}
	if aliases != 0 {
		t.Fatalf("aliases=%d", aliases)
	}
}
