package privacy

import (
	"context"
	"crypto/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
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
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_ledger_outbox WHERE tombstone_id IN (SELECT id FROM ttg.privacy_tombstones WHERE actor_ref_protected = $1)`, actor)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_erase_requests WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanup, `DELETE FROM ttg.privacy_tombstones WHERE actor_ref_protected = $1`, actor)
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
	proof, err := ledger.Verify("db-test", time.Now().UTC())
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
