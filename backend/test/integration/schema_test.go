package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// schemaTables is the authoritative table list from docs/plan/04_DATABASE.md.
var schemaTables = []string{
	"villages", "sessions", "players", "session_player_progress", "wave_members",
	"ballots", "input_receipts", "input_events", "checkpoints", "domain_events",
	"stream_outbox", "operator_accounts", "operator_sessions", "operator_audit",
	"writer_leases", "admin_idempotency_keys", "privacy_tombstones",
	"privacy_erase_requests", "broadcaster_source_bindings",
}

func schemaDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TTG_TEST_DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("TTG_TEST_DATABASE_URL is not set; run tooling/migrate-up.ps1 against the development database")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return db
}

var uuidCounter atomic.Uint64

func newUUID(t *testing.T) string {
	t.Helper()
	// UUIDv4 shaped value; collisions are irrelevant because every test rolls
	// its transaction back, but the counter keeps multiple calls distinct.
	now := time.Now().UnixNano() + int64(uuidCounter.Add(1))
	return fmt.Sprintf("%08x-%04x-4%03x-8%03x-%012x",
		uint32(now>>32), uint16(now>>16), uint16(now)&0x0fff, uint16(now>>8)&0x0fff, now&0xffffffffffff)
}

func TestSchemaContractTables(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()

	rows, err := db.QueryContext(ctx,
		`SELECT table_name FROM information_schema.tables WHERE table_schema = 'ttg'`)
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer func() { _ = rows.Close() }()
	got := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	for _, want := range schemaTables {
		if !got[want] {
			t.Errorf("missing table ttg.%s", want)
		}
	}
	// The goose bookkeeping table must live in the owned schema, never in public.
	if got["goose_db_version"] {
		t.Log("goose_db_version present in schema ttg (expected)")
	}
	var publicTables int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'`).Scan(&publicTables); err != nil {
		t.Fatalf("public tables: %v", err)
	}
	if publicTables != 0 {
		t.Errorf("schema public contains %d table(s); migrations must stay in ttg", publicTables)
	}
}

func TestSchemaContractIndexesAndConstraints(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()

	var uniquePartial string
	if err := db.QueryRowContext(ctx, `
		SELECT indexdef FROM pg_indexes
		WHERE schemaname = 'ttg' AND indexname = 'sessions_one_active_per_village'`).Scan(&uniquePartial); err != nil {
		t.Fatalf("missing sessions_one_active_per_village: %v", err)
	}
	if !strings.Contains(uniquePartial, "UNIQUE") || !strings.Contains(uniquePartial, "WHERE") {
		t.Errorf("sessions_one_active_per_village must be a UNIQUE partial index, got %q", uniquePartial)
	}

	var pending string
	if err := db.QueryRowContext(ctx, `
		SELECT indexdef FROM pg_indexes
		WHERE schemaname = 'ttg' AND indexname = 'input_events_pending'`).Scan(&pending); err != nil {
		t.Fatalf("missing input_events_pending: %v", err)
	}
	if !strings.Contains(pending, "applied_at_tick IS NULL") {
		t.Errorf("input_events_pending must be partial on applied_at_tick IS NULL, got %q", pending)
	}

	for _, constraint := range []string{
		"villages_wall_hp_within_tier",
		"players_village_actor_key",
		"input_receipts_unique_event",
	} {
		var n int
		if err := db.QueryRowContext(ctx, `
			SELECT count(*) FROM pg_constraint c
			JOIN pg_namespace n ON n.oid = c.connamespace
			WHERE n.nspname = 'ttg' AND c.conname = $1`, constraint).Scan(&n); err != nil {
			t.Fatalf("constraint query: %v", err)
		}
		if n != 1 {
			t.Errorf("constraint %s not found", constraint)
		}
	}
}

// TestSchemaRejectsInvalidRows exercises the documented CHECK and uniqueness
// rules inside a transaction that is always rolled back.
func TestSchemaRejectsInvalidRows(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	mustFail := func(name, query string, args ...any) {
		t.Helper()
		// Each expected failure aborts the transaction, so isolate it behind a
		// savepoint and roll back to it afterwards.
		if _, err := tx.ExecContext(ctx, "SAVEPOINT expected_failure"); err != nil {
			t.Fatalf("%s: savepoint: %v", name, err)
		}
		_, err := tx.ExecContext(ctx, query, args...)
		if err == nil {
			t.Fatalf("%s: expected the schema to reject the row", name)
		}
		wantErr := regexp.MustCompile(`violates|duplicate key|invalid input`)
		if !wantErr.MatchString(err.Error()) {
			t.Fatalf("%s: unexpected error %v", name, err)
		}
		if _, rbErr := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT expected_failure"); rbErr != nil {
			t.Fatalf("%s: rollback to savepoint: %v", name, rbErr)
		}
	}

	villageID := newUUID(t)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ttg.villages (id, broadcaster_key) VALUES ($1, $2)`,
		villageID, strings.Repeat("a", 64)); err != nil {
		t.Fatalf("insert village: %v", err)
	}

	mustFail("wall_hp above the tier maximum", `
		INSERT INTO ttg.villages (id, broadcaster_key, wall_tier, wall_hp)
		VALUES ($1, $2, 1, 1200)`, newUUID(t), strings.Repeat("b", 64))
	mustFail("consecutive_wins above 10", `
		INSERT INTO ttg.villages (id, broadcaster_key, consecutive_wins)
		VALUES ($1, $2, 11)`, newUUID(t), strings.Repeat("c", 64))
	mustFail("duplicate broadcaster_key", `
		INSERT INTO ttg.villages (id, broadcaster_key) VALUES ($1, $2)`,
		newUUID(t), strings.Repeat("a", 64))

	runningID := newUUID(t)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version)
		VALUES ($1, $2, 'mock', 'running', 'ttg-rules-v1')`, runningID, villageID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	mustFail("second non-stopped session for the same village", `
		INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version)
		VALUES ($1, $2, 'mock', 'waiting', 'ttg-rules-v1')`, newUUID(t), villageID)
	mustFail("unknown lifecycle", `
		INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version)
		VALUES ($1, $2, 'mock', 'exploded', 'ttg-rules-v1')`, newUUID(t), villageID)
	mustFail("wave_tick outside -1..3000", `
		INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, wave_tick, rules_version)
		VALUES ($1, $2, 'mock', 'waiting', 3001, 'ttg-rules-v1')`, newUUID(t), villageID)

	mustFail("unknown role", `
		INSERT INTO ttg.wave_members (session_id, wave_no, actor_key, role, joined_tick)
		VALUES ($1, 1, $2, 'wizard', 0)`, runningID, strings.Repeat("d", 64))
	mustFail("negative joined_tick", `
		INSERT INTO ttg.wave_members (session_id, wave_no, actor_key, role, joined_tick)
		VALUES ($1, 1, $2, 'builder', -1)`, runningID, strings.Repeat("e", 64))

	mustFail("unknown ballot option", `
		INSERT INTO ttg.ballots (session_id, wave_no, actor_key, option, input_seq)
		VALUES ($1, 2, $2, 'D', 1)`, runningID, strings.Repeat("f", 64))

	mustFail("unknown input kind", `
		INSERT INTO ttg.input_events
			(session_id, input_seq, accepted_tick, scheduled_tick, wave_no, actor_key, kind, canonical_payload, rules_version)
		VALUES ($1, 1, 0, 1, 1, $2, 'gossip', '{}'::jsonb, 'ttg-rules-v1')`, runningID, strings.Repeat("g", 64))

	mustFail("session_xp above 300", `
		INSERT INTO ttg.session_player_progress (session_id, actor_key, session_xp)
		VALUES ($1, $2, 301)`, runningID, strings.Repeat("h", 64))

	mustFail("alias longer than 256 bytes", `
		INSERT INTO ttg.players (id, village_id, actor_key, ephemeral, alias)
		VALUES ($1, $2, $3, FALSE, $4)`, newUUID(t), villageID, strings.Repeat("i", 64), strings.Repeat("x", 257))

	// A valid pending input row and the partial index it depends on.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ttg.input_events
			(session_id, input_seq, accepted_tick, scheduled_tick, wave_no, actor_key, kind, canonical_payload, rules_version)
		VALUES ($1, 2, 10, 11, 1, $2, 'role', '{"role":"builder"}'::jsonb, 'ttg-rules-v1')`,
		runningID, strings.Repeat("j", 64)); err != nil {
		t.Fatalf("valid input_events row rejected: %v", err)
	}

	// Stopped sessions do not conflict with the partial unique index.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version)
		VALUES ($1, $2, 'mock', 'stopped', 'ttg-rules-v1')`, newUUID(t), villageID); err != nil {
		t.Fatalf("stopped session should be allowed alongside an active one: %v", err)
	}
}

// TestSchemaSnapshotMatchesDatabase guards the sqlc input: backend/db/schema.sql
// must describe exactly the tables the migrations create. Column-level drift is
// not detected here (it needs pg_dump); run tooling/schema-dump.ps1|sh after
// changing a migration and commit the regenerated snapshot.
func TestSchemaSnapshotMatchesDatabase(t *testing.T) {
	db := schemaDB(t)
	ctx := context.Background()

	raw, err := os.ReadFile("../../db/schema.sql")
	if err != nil {
		t.Fatalf("read schema snapshot: %v", err)
	}
	declared := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^CREATE TABLE ttg\.([a-z_]+)`).FindAllStringSubmatch(string(raw), -1) {
		declared[m[1]] = true
	}
	if len(declared) == 0 {
		t.Fatal("schema.sql declares no tables; regenerate it with tooling/schema-dump")
	}

	rows, err := db.QueryContext(ctx,
		`SELECT table_name FROM information_schema.tables WHERE table_schema = 'ttg' AND table_name <> 'goose_db_version'`)
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer func() { _ = rows.Close() }()
	actual := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		actual[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	for name := range declared {
		if !actual[name] {
			t.Errorf("schema.sql declares ttg.%s but the migrated database does not have it; regenerate the snapshot", name)
		}
	}
	for name := range actual {
		if !declared[name] {
			t.Errorf("migrated table ttg.%s is missing from schema.sql; run tooling/schema-dump and sqlc generate", name)
		}
	}
}
